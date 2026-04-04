package globalflow

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// CommandDeps runs voice-routed shell commands via the host (single command directive per turn).
type CommandDeps struct {
	HostApply interface {
		ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
	}
	ExtensionHost interface {
		ReadHostFile(path string) (string, error)
	}
	EditModel  agent.ModelClient
	NewBatchID func() string
}

var commandContextProbeFiles = []string{
	"package.json",
	"pnpm-lock.yaml",
	"yarn.lock",
	"package-lock.json",
	"bun.lock",
	"go.mod",
	"Cargo.toml",
	"pyproject.toml",
	"requirements.txt",
	"poetry.lock",
	"uv.lock",
}

const (
	commandContextMaxTotalBytes = 96_000
	commandContextMaxFileBytes  = 24_000
	defaultCommandTimeoutMs     = int64(300_000)
)

var allowedShellBasenames = map[string]struct{}{
	"cmd": {}, "cmd.exe": {},
	"powershell": {}, "powershell.exe": {},
	"pwsh": {}, "pwsh.exe": {},
	"sh": {}, "bash": {}, "zsh": {}, "fish": {},
}

func isAllowedShellCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(cmd))
	_, ok := allowedShellBasenames[base]
	return ok
}

func truncateUTF8(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	s = s[:maxBytes]
	for len(s) > 0 && !utf8RuneStart(s[len(s)-1]) {
		s = s[:len(s)-1]
	}
	return s + "\n…(truncated)…"
}

func utf8RuneStart(b byte) bool {
	return b&0xC0 != 0x80
}

func gatherWorkspaceCommandContext(host interface {
	ReadHostFile(path string) (string, error)
}, root string) (bundle string, foundNames []string) {
	root = strings.TrimSpace(root)
	if root == "" || host == nil {
		return "", nil
	}
	var b strings.Builder
	total := 0
	for _, name := range commandContextProbeFiles {
		p := filepath.Join(root, name)
		body, err := host.ReadHostFile(p)
		if err != nil || strings.TrimSpace(body) == "" {
			continue
		}
		foundNames = append(foundNames, name)
		chunk := truncateUTF8(body, commandContextMaxFileBytes)
		header := fmt.Sprintf("--- %s ---\n", name)
		add := header + chunk
		if !strings.HasSuffix(add, "\n") {
			add += "\n"
		}
		if total+len(add) > commandContextMaxTotalBytes {
			break
		}
		b.WriteString(add)
		total += len(add)
	}
	return b.String(), foundNames
}

// nearestPackageJSONRoot walks up from activeFile's directory looking for package.json (non-empty).
func nearestPackageJSONRoot(host interface {
	ReadHostFile(path string) (string, error)
}, activeFile string) string {
	activeFile = strings.TrimSpace(activeFile)
	if activeFile == "" || host == nil {
		return ""
	}
	dir := filepath.Dir(filepath.Clean(activeFile))
	for {
		if dir == "" || dir == "." {
			break
		}
		p := filepath.Join(dir, "package.json")
		body, err := host.ReadHostFile(p)
		if err == nil && strings.TrimSpace(body) != "" {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

type voiceCommandPlan struct {
	Status             string   `json:"status"`
	Message            string   `json:"message"`
	Command            string   `json:"command"`
	Args               []string `json:"args"`
	WorkingDirectory   string   `json:"workingDirectory"`
	TimeoutMs          *int64   `json:"timeoutMs"`
	Detached           *bool    `json:"detached"`
}

// voiceCommandResponseSchema must be a JSON object at the root with type "object".
// OpenAI response_format json_schema rejects root-level oneOf (no root type → API error "type: None").
func voiceCommandResponseSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"status"},
		"properties": map[string]any{
			"status": map[string]any{
				"type":        "string",
				"enum":        []string{"ok", "insufficient_context"},
				"description": `Use "ok" with command and args, or "insufficient_context" with message only.`,
			},
			"message": map[string]any{
				"type":        "string",
				"description": "When status is insufficient_context: short reason for the user (no markdown). Omit when status is ok.",
			},
			"command": map[string]any{
				"type": "string",
				"description": "When status is ok: ONLY a shell executable — powershell.exe, pwsh, cmd.exe, bash, sh, zsh, or fish. " +
					"INVALID (host will reject): npx, npm, pnpm, yarn, node, git, cargo, go, python.",
			},
			"args": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
				"description": "When status is ok: arguments for that shell only. Put the full dev invocation inside -Command (win32) or -c / -lc (Unix) — " +
					"e.g. args [\"-NoProfile\",\"-NonInteractive\",\"-Command\",\"npx create-expo-app@latest myapp\"].",
			},
			"workingDirectory": map[string]any{
				"type":        "string",
				"description": "Optional absolute cwd when status is ok.",
			},
			"timeoutMs": map[string]any{
				"type":        "integer",
				"description": "Optional timeout in ms when status is ok. Ignored when detached is true.",
			},
			"detached": map[string]any{
				"type":        "boolean",
				"description": `When true: host opens an integrated terminal tab (user sees output, Ctrl+C stops); no exit wait. Use for dev servers: "npx expo start", "npm run dev", "next dev", vite. For PowerShell do NOT pass -NonInteractive when detached (user needs an interactive TTY). Omit or false for one-shot commands.`,
			},
		},
	}
}

// HandleCommand plans and runs one shell command from voice (host applies CommandDirective).
func HandleCommand(
	deps *CommandDeps,
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	transcript string,
) (protocol.VoiceTranscriptCompletion, string) {
	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command: empty transcript"
	}
	if deps == nil || deps.HostApply == nil || deps.ExtensionHost == nil || deps.EditModel == nil || deps.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command: host or model not configured"
	}

	root := strings.TrimSpace(params.WorkspaceRoot)
	ctxBody, probeFound := gatherWorkspaceCommandContext(deps.ExtensionHost, root)
	hasRootPackageJSON := false
	for _, n := range probeFound {
		if n == "package.json" {
			hasRootPackageJSON = true
			break
		}
	}
	nearPkg := nearestPackageJSONRoot(deps.ExtensionHost, params.ActiveFile)

	sys := strings.TrimSpace(`You are Vocode's voice command planner inside an IDE.
The host runs your JSON by spawning EXACTLY ONE process: the string in "command" must be a shell binary only. It will REJECT npx, npm, pnpm, node, git, cargo, etc. as "command".

Output ONE JSON object matching the schema.

status "insufficient_context": set "message"; leave command/args empty or omit.

status "ok": REQUIRED SHAPE
- "command": one of powershell.exe, pwsh, cmd.exe, bash, sh, zsh, fish (no other values).
- "args": that shell's argv. The dev tools (npx, npm, pnpm, …) appear ONLY inside the script string passed to -Command (Windows) or -c / -lc (macOS/Linux), never as "command".

Concrete win32 example (Expo scaffold):
  "command": "powershell.exe",
  "args": ["-NoProfile","-NonInteractive","-Command","npx create-expo-app@latest VocodedApp --yes"]

Concrete Unix example:
  "command": "bash",
  "args": ["-lc","npx create-expo-app@latest VocodedApp --yes"]

WRONG (will fail): "command":"npx","args":["create-expo-app",...]

Rules:
- Pick runners (npx vs pnpm dlx, etc.) inside the script; honor the user's tool if they name one.
- workingDirectory: MUST be absolute. Prefer the host hint nearestPackageJsonRootFromActiveFile when it is non-empty and the command is app-local (expo, react-native, vite, next dev, npm run dev) — especially when workspace root has no package.json but that hint path does (monorepo / app in subfolder). Otherwise default to workspace root for installs/scaffolds.
- detached: set true for commands that keep running until the user stops them (expo start, dev servers); the host opens an integrated terminal so the user can see logs and press Ctrl+C. Do not use PowerShell -NonInteractive when detached. Set false or omit for one-shot commands (install, build, test, create-app).
- Greenfield: user may name pnpm/npx explicitly — still wrap in a shell as above.
- Vague scaffold + no repo files: pick a conventional script (e.g. npx create-expo-app@latest); stderr shows if it fails.
- "install dependencies" with no package.json/lockfile and no tool named → insufficient_context.
- Never markdown. JSON only.`)

	var userPayload strings.Builder
	userPayload.WriteString("User transcript:\n")
	userPayload.WriteString(transcript)
	userPayload.WriteString("\n\nhostPlatform: ")
	hp := strings.TrimSpace(params.HostPlatform)
	if hp == "" {
		hp = "(unknown)"
	}
	userPayload.WriteString(hp)
	userPayload.WriteString("\nworkspaceRoot: ")
	if root != "" {
		userPayload.WriteString(root)
	} else {
		userPayload.WriteString("(empty)")
	}
	userPayload.WriteString("\nactiveFile: ")
	userPayload.WriteString(strings.TrimSpace(params.ActiveFile))
	userPayload.WriteString("\nworkspaceRootHasPackageJson: ")
	if hasRootPackageJSON {
		userPayload.WriteString("true")
	} else {
		userPayload.WriteString("false")
	}
	userPayload.WriteString("\nnearestPackageJsonRootFromActiveFile: ")
	if nearPkg != "" {
		userPayload.WriteString(nearPkg)
	} else {
		userPayload.WriteString("(none — walk upward from activeFile found no package.json)")
	}
	if nearPkg != "" && root != "" && !strings.EqualFold(filepath.Clean(nearPkg), filepath.Clean(root)) {
		userPayload.WriteString("\nNote: nearest package.json is NOT at workspace root; for Expo/Node app commands prefer workingDirectory=")
		userPayload.WriteString(nearPkg)
		userPayload.WriteString(" unless the transcript clearly refers to the repo root.")
	}
	userPayload.WriteString("\n")
	if ctxBody != "" {
		userPayload.WriteString("\nGathered workspace file excerpts:\n")
		userPayload.WriteString(ctxBody)
	}

	raw, err := deps.EditModel.Call(context.Background(), agent.CompletionRequest{
		System:     sys,
		User:       userPayload.String(),
		JSONSchema: voiceCommandResponseSchema(),
	})
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command model: " + err.Error()
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command: empty model response"
	}

	var plan voiceCommandPlan
	if err := json.Unmarshal([]byte(raw), &plan); err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command: invalid model JSON"
	}

	switch strings.ToLower(strings.TrimSpace(plan.Status)) {
	case "insufficient_context":
		msg := strings.TrimSpace(plan.Message)
		if msg == "" {
			msg = "Not enough context to run a command."
		}
		return protocol.VoiceTranscriptCompletion{
			Success: false,
			Summary: msg,
		}, msg
	case "ok":
		// continue
	default:
		return protocol.VoiceTranscriptCompletion{Success: false}, `command: model status must be "ok" or "insufficient_context"`
	}

	cmd := strings.TrimSpace(plan.Command)
	if !isAllowedShellCommand(cmd) {
		return protocol.VoiceTranscriptCompletion{Success: false},
			`command: "command" must be a shell (powershell.exe, bash, sh, …) — put npx/npm/pnpm inside -Command or -c, not in "command"`
	}
	if len(plan.Args) == 0 {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command: model returned empty args"
	}

	wd := strings.TrimSpace(plan.WorkingDirectory)
	if wd == "" && root != "" {
		wd = root
	}
	if wd != "" && !filepath.IsAbs(wd) {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command: workingDirectory must be absolute"
	}

	detached := plan.Detached != nil && *plan.Detached

	var timeout *int64
	if !detached {
		timeout = plan.TimeoutMs
		if timeout == nil || *timeout <= 0 {
			t := defaultCommandTimeoutMs
			timeout = &t
		}
	}

	dir := protocol.CommandDirective{
		Command:          cmd,
		Args:             append([]string(nil), plan.Args...),
		WorkingDirectory: wd,
		Detached:         detached,
		TimeoutMs:        timeout,
	}

	summaryLine := cmd
	if len(dir.Args) > 0 {
		summaryLine += " " + strings.Join(dir.Args, " ")
	}
	if len(summaryLine) > 200 {
		summaryLine = summaryLine[:200] + "…"
	}

	batchID := deps.NewBatchID()
	pending := &session.DirectiveApplyBatch{ID: batchID, NumDirectives: 1}
	if vs != nil {
		vs.PendingDirectiveApply = pending
	}
	hostRes, err := deps.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: batchID,
		ActiveFile:   params.ActiveFile,
		Directives: []protocol.VoiceTranscriptDirective{
			{
				Kind:             "command",
				CommandDirective: &dir,
			},
		},
	})
	if err != nil {
		if vs != nil {
			vs.PendingDirectiveApply = nil
		}
		return protocol.VoiceTranscriptCompletion{Success: false}, "command apply: " + err.Error()
	}
	if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
		if vs != nil {
			vs.PendingDirectiveApply = nil
		}
		em := err.Error()
		return protocol.VoiceTranscriptCompletion{
			Success: false,
			Summary: em,
		}, em
	}
	if vs != nil {
		vs.PendingDirectiveApply = nil
	}

	summaryVerb := "Ran"
	if detached {
		summaryVerb = "Started"
	}
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       summaryVerb + ": " + summaryLine,
		UiDisposition: "hidden",
	}, ""
}
