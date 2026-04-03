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

type voiceCommandPlan struct {
	Status             string   `json:"status"`
	Message            string   `json:"message"`
	Command            string   `json:"command"`
	Args               []string `json:"args"`
	WorkingDirectory   string   `json:"workingDirectory"`
	TimeoutMs          *int64   `json:"timeoutMs"`
}

func voiceCommandResponseSchema() map[string]any {
	return map[string]any{
		"oneOf": []any{
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"status", "command", "args"},
				"properties": map[string]any{
					"status": map[string]any{
						"type": "string",
						"enum": []string{"ok"},
					},
					"command": map[string]any{
						"type":        "string",
						"description": "Single shell executable only: cmd, powershell, pwsh, sh, bash, zsh, or fish (basename or with .exe on Windows).",
					},
					"args": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"workingDirectory": map[string]any{
						"type":        "string",
						"description": "Absolute directory for spawn cwd; use workspace root when appropriate.",
					},
					"timeoutMs": map[string]any{
						"type":        "integer",
						"description": "Optional timeout in ms; omit to use daemon default.",
					},
				},
			},
			map[string]any{
				"type":                 "object",
				"additionalProperties": false,
				"required":             []string{"status", "message"},
				"properties": map[string]any{
					"status": map[string]any{
						"type": "string",
						"enum": []string{"insufficient_context"},
					},
					"message": map[string]any{
						"type":        "string",
						"description": "Short reason for the user (no markdown fences).",
					},
				},
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
	ctxBody, _ := gatherWorkspaceCommandContext(deps.ExtensionHost, root)

	sys := strings.TrimSpace(`You are Vocode's voice command planner inside an IDE.
The user wants a shell command executed. You output ONE JSON object matching the schema.

Rules:
- status "ok": emit a real command as a trusted shell + args (non-interactive).
  - Windows (hostPlatform win32): prefer powershell.exe with -NoProfile, -NonInteractive, -Command, and ONE string containing the full script, OR cmd.exe with /c and one string.
  - macOS/Linux: prefer bash with -lc and ONE string for the script, or sh -c.
- The executable in "command" must be ONLY a shell from this set: cmd.exe, powershell, pwsh, sh, bash, zsh, fish (single token, no spaces).
- Put dev tools (pnpm, npm, npx, pnpm dlx, yarn, yarn dlx, bun, cargo, go, etc.) inside the script string passed to -Command or -c, not as "command".
- You choose which runner and subcommands to use from the gathered workspace files, stack conventions, and hostPlatform — unless the user explicitly names a tool or command (e.g. "with pnpm", "use npx"), in which case follow their choice.
- workingDirectory: absolute path. Use the workspace root when running installs or project commands. For monorepos, you may use a package directory if the active file path implies it and the user intent is local to that package.
- If the user names a tool explicitly (pnpm, npx, …), you MUST honor it even when package.json is missing (greenfield).
- If the user is vague about scaffolding (e.g. "create an expo app") and there are no repo signals, emit a conventional one-shot command and pick a reasonable runner yourself (npx, pnpm dlx, etc.); the host will surface stderr if it fails.
- If the user asks to install dependencies but there is no package.json or Node lockfile in the gathered context AND they did not name a package manager or tool: status MUST be "insufficient_context" with a short message (ask them to name a tool or open the project folder).
- status "insufficient_context": when you cannot responsibly choose a command (ambiguous with no evidence and no tool named, or missing critical info). Include message only; no command/args.
- Never emit markdown. JSON only.`)

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
		}, ""
	case "ok":
		// continue
	default:
		return protocol.VoiceTranscriptCompletion{Success: false}, `command: model status must be "ok" or "insufficient_context"`
	}

	cmd := strings.TrimSpace(plan.Command)
	if !isAllowedShellCommand(cmd) {
		return protocol.VoiceTranscriptCompletion{Success: false}, "command: model returned a non-shell executable"
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

	timeout := plan.TimeoutMs
	if timeout == nil || *timeout <= 0 {
		t := defaultCommandTimeoutMs
		timeout = &t
	}

	dir := protocol.CommandDirective{
		Command:            cmd,
		Args:               append([]string(nil), plan.Args...),
		WorkingDirectory:   wd,
		TimeoutMs:          timeout,
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
		return protocol.VoiceTranscriptCompletion{
			Success: false,
			Summary: err.Error(),
		}, ""
	}
	if vs != nil {
		vs.PendingDirectiveApply = nil
	}

	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "Ran: " + summaryLine,
		UiDisposition: "hidden",
	}, ""
}
