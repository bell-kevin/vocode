package transcript

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agent"
	"vocoding.net/vocode/v2/apps/daemon/internal/dispatch"
	"vocoding.net/vocode/v2/apps/daemon/internal/edits"
	"vocoding.net/vocode/v2/apps/daemon/internal/intent"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// TranscriptService adapts the voice.transcript RPC to the agent runtime and
// action-plan execution (edits via structured results for the extension;
// commands on the daemon).
type TranscriptService struct {
	agent    *agent.Agent
	dispatch *dispatch.Dispatcher
	symbols  symbols.Resolver

	queue chan transcriptJob

	coalesceWindow time.Duration
	maxMergeJobs   int
	maxMergeChars  int
	maxPlannerTurns int
	maxContextRounds int
	maxContextBytes int
	maxConsecutiveContextReq int

	startOnce sync.Once
}

type transcriptJob struct {
	params protocol.VoiceTranscriptParams
	resp   chan transcriptAcceptResp
}

type transcriptAcceptResp struct {
	result protocol.VoiceTranscriptResult
	ok     bool
}

func NewService(
	agentRuntime *agent.Agent,
	dispatch *dispatch.Dispatcher,
) *TranscriptService {
	queueSize := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_QUEUE_SIZE", 10)
	coalesceMs := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_COALESCE_MS", 750)
	maxMergeJobs := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_JOBS", 5)
	maxMergeChars := envInt("VOCODE_DAEMON_VOICE_TRANSCRIPT_MAX_MERGE_CHARS", 6000)
	maxPlannerTurns := envInt("VOCODE_DAEMON_VOICE_MAX_PLANNER_TURNS", 8)
	maxContextRounds := envInt("VOCODE_DAEMON_VOICE_MAX_CONTEXT_ROUNDS", 2)
	maxContextBytes := envInt("VOCODE_DAEMON_VOICE_MAX_CONTEXT_BYTES", 12000)
	maxConsecutiveContextReq := envInt("VOCODE_DAEMON_VOICE_MAX_CONSECUTIVE_CONTEXT_REQUESTS", 3)

	s := &TranscriptService{
		agent:          agentRuntime,
		dispatch:       dispatch,
		symbols:        symbols.NewTreeSitterResolver(),
		coalesceWindow: time.Duration(coalesceMs) * time.Millisecond,
		maxMergeJobs:   maxMergeJobs,
		maxMergeChars:  maxMergeChars,
		maxPlannerTurns: maxPlannerTurns,
		maxContextRounds: maxContextRounds,
		maxContextBytes: maxContextBytes,
		maxConsecutiveContextReq: maxConsecutiveContextReq,
	}

	if queueSize > 0 {
		s.queue = make(chan transcriptJob, queueSize)
		s.startOnce.Do(func() {
			go s.runWorker()
		})
	}

	return s
}

func (s *TranscriptService) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptResult, bool) {
	params.Text = strings.TrimSpace(params.Text)
	if params.Text == "" {
		return protocol.VoiceTranscriptResult{}, false
	}

	// If the queue is disabled, execute immediately.
	if s.queue == nil {
		return s.acceptTranscriptDirect(params)
	}

	respCh := make(chan transcriptAcceptResp, 1)
	job := transcriptJob{
		params: params,
		resp:   respCh,
	}

	// Keep RPC calls responsive: if the queue is full, succeed with a planError
	// so the extension can ignore it without crashing the session.
	select {
	case s.queue <- job:
	default:
		job.resp <- transcriptAcceptResp{
			result: protocol.VoiceTranscriptResult{
				Accepted:  true,
				PlanError: "transcript queue full",
			},
			ok: true,
		}
	}

	resp := <-respCh
	return resp.result, resp.ok
}

func (s *TranscriptService) runWorker() {
	// buffered contains jobs that couldn't be merged into the current coalescing batch,
	// but must preserve FIFO order for later processing.
	buffered := make([]transcriptJob, 0, cap(s.queue))

	for {
		primary := func() transcriptJob {
			if len(buffered) > 0 {
				j := buffered[0]
				buffered = buffered[1:]
				return j
			}
			return <-s.queue
		}()

		baseActiveFile := strings.TrimSpace(primary.params.ActiveFile)

		batch := []transcriptJob{primary}
		mergedTextParts := []string{primary.params.Text}
		mergedChars := len(primary.params.Text)

		timer := time.NewTimer(s.coalesceWindow)

		for collecting := true; collecting; {
			select {
			case j := <-s.queue:
				activeFile := strings.TrimSpace(j.params.ActiveFile)
				text := strings.TrimSpace(j.params.Text)
				if text == "" {
					// Invalid/empty transcripts should just no-op.
					j.resp <- transcriptAcceptResp{
						result: protocol.VoiceTranscriptResult{Accepted: true},
						ok:     true,
					}
					continue
				}

				eligible := activeFile == baseActiveFile &&
					len(batch) < s.maxMergeJobs &&
					mergedChars+1+len(text) <= s.maxMergeChars

				if eligible {
					j.params.Text = text
					batch = append(batch, j)
					mergedTextParts = append(mergedTextParts, text)
					mergedChars += 1 + len(text)
				} else {
					buffered = append(buffered, j)
				}
			case <-timer.C:
				collecting = false
			}
		}

		timer.Stop()

		mergedParams := primary.params
		mergedParams.Text = strings.Join(mergedTextParts, " ")

		mergedResult, ok := s.acceptTranscriptDirect(mergedParams)

		// Respond: only the primary job returns the actual execution.
		// Coalesced jobs succeed with an empty result to avoid duplicate UI edits.
		for i, j := range batch {
			if i == 0 {
				j.resp <- transcriptAcceptResp{
					result: mergedResult,
					ok:     ok,
				}
			} else {
				j.resp <- transcriptAcceptResp{
					result: protocol.VoiceTranscriptResult{Accepted: true},
					ok:     true,
				}
			}
		}
	}
}

func (s *TranscriptService) acceptTranscriptDirect(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptResult, bool) {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptResult{}, false
	}

	maxTurns := s.maxPlannerTurns
	if maxTurns <= 0 {
		maxTurns = 8
	}
	turnCtx := agent.PlanningContext{}
	completed := make([]intent.NextIntent, 0, maxTurns)
	contextRounds := 0
	consecutiveContextReq := 0
	editCounter := 0
	stepResults := make([]protocol.VoiceTranscriptStepResult, 0, maxTurns)
	stopPlanning := false

	for i := 0; i < maxTurns; i++ {
		next, err := s.agent.NextIntent(context.Background(), agent.ModelInput{
			Transcript:       text,
			CompletedActions: append([]intent.NextIntent(nil), completed...),
			Context:          turnCtx,
		})
		if err != nil {
			return protocol.VoiceTranscriptResult{
				Accepted:  true,
				PlanError: err.Error(),
			}, true
		}
		if err := intent.ValidateNextIntent(next); err != nil {
			return protocol.VoiceTranscriptResult{
				Accepted:  true,
				PlanError: err.Error(),
			}, true
		}
		if next.Kind == intent.NextIntentKindDone {
			break
		}
		if next.Kind == intent.NextIntentKindRequestContext {
			contextRounds++
			consecutiveContextReq++
			if s.maxContextRounds > 0 && contextRounds > s.maxContextRounds {
				return protocol.VoiceTranscriptResult{
					Accepted:  true,
					PlanError: "max context rounds reached",
				}, true
			}
			if s.maxConsecutiveContextReq > 0 && consecutiveContextReq > s.maxConsecutiveContextReq {
				return protocol.VoiceTranscriptResult{
					Accepted:  true,
					PlanError: "too many consecutive context requests",
				}, true
			}
			updated, err := s.fulfillContextRequest(params, turnCtx, next.RequestContext)
			if err != nil {
				return protocol.VoiceTranscriptResult{
					Accepted:  true,
					PlanError: err.Error(),
				}, true
			}
			if s.maxContextBytes > 0 && estimatePlanningContextBytes(updated) > s.maxContextBytes {
				return protocol.VoiceTranscriptResult{
					Accepted:  true,
					PlanError: "context budget exceeded",
				}, true
			}
			turnCtx = updated
			completed = append(completed, next)
			continue
		}
		consecutiveContextReq = 0
		editCtx, planErr := buildEditExecutionContext(params, next)
		if planErr != "" {
			return protocol.VoiceTranscriptResult{
				Accepted:  true,
				PlanError: planErr,
			}, true
		}
		st, err := s.dispatch.ExecuteNextIntent(next, editCtx)
		if err != nil {
			return protocol.VoiceTranscriptResult{
				Accepted:  true,
				PlanError: err.Error(),
			}, true
		}
		switch {
		case st.EditResult != nil:
			if st.EditResult.Kind == "success" {
				for i := range st.EditResult.Actions {
					if st.EditResult.Actions[i].EditId == "" {
						st.EditResult.Actions[i].EditId = fmt.Sprintf("edit-%d", editCounter)
						editCounter++
					}
				}
			}
			stepResults = append(stepResults, protocol.VoiceTranscriptStepResult{
				Kind:       "edit",
				EditResult: st.EditResult,
			})
			completed = append(completed, next)
			if st.EditResult.Kind == "failure" {
				stopPlanning = true
			}
		case st.CommandParams != nil:
			stepResults = append(stepResults, protocol.VoiceTranscriptStepResult{
				Kind:          "run_command",
				CommandParams: st.CommandParams,
			})
			completed = append(completed, next)
		case st.Navigation != nil:
			stepResults = append(stepResults, protocol.VoiceTranscriptStepResult{
				Kind:             "navigate",
				NavigationIntent: toProtocolNavigationIntent(*st.Navigation),
			})
			completed = append(completed, next)
		}
		if stopPlanning {
			break
		}
	}
	if len(completed) >= maxTurns {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: "max planner turns reached",
		}, true
	}

	result := protocol.VoiceTranscriptResult{
		Accepted: true,
		Steps:    stepResults,
	}
	if err := result.Validate(); err != nil {
		return protocol.VoiceTranscriptResult{
			Accepted:  true,
			PlanError: err.Error(),
		}, true
	}
	return result, true
}

func (s *TranscriptService) fulfillContextRequest(
	params protocol.VoiceTranscriptParams,
	in agent.PlanningContext,
	req *intent.RequestContextIntent,
) (agent.PlanningContext, error) {
	if req == nil {
		return in, fmt.Errorf("request_context missing payload")
	}
	out := in
	switch req.Kind {
	case intent.RequestContextKindSymbols:
		query := strings.TrimSpace(req.Query)
		if query == "" {
			return out, fmt.Errorf("request_symbols requires query")
		}
		if s.symbols == nil {
			return out, fmt.Errorf("symbol resolver unavailable")
		}
		matches, err := s.symbols.ResolveSymbol(strings.TrimSpace(params.WorkspaceRoot), query, "", strings.TrimSpace(params.ActiveFile))
		if err != nil {
			return out, err
		}
		limit := clampContextMax(req.MaxResult, 10)
		if out.Symbols == nil {
			out.Symbols = make([]symbols.SymbolRef, 0, limit)
		}
		seen := map[string]bool{}
		for _, sref := range out.Symbols {
			seen[sref.ID] = true
		}
		for _, m := range matches {
			if m.ID == "" || seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			out.Symbols = append(out.Symbols, m)
			if len(out.Symbols) >= limit {
				break
			}
		}
		return out, nil
	case intent.RequestContextKindFileExcerpt:
		target := strings.TrimSpace(req.Path)
		ec := edits.EditExecutionContext{
			ActiveFile:    params.ActiveFile,
			WorkspaceRoot: params.WorkspaceRoot,
		}
		path := ec.ResolvePath(target)
		if path == "" {
			return out, fmt.Errorf("request_file_excerpt requires resolvable path")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return out, err
		}
		content := string(b)
		const maxChars = 4000
		if len(content) > maxChars {
			content = content[:maxChars]
		}
		out.Excerpts = append(out.Excerpts, agent.FileExcerpt{Path: filepath.Clean(path), Content: content})
		return out, nil
	case intent.RequestContextKindUsages:
		ref, err := symbols.ParseSymbolID(strings.TrimSpace(req.SymbolID))
		if err != nil {
			return out, fmt.Errorf("request_usages requires valid symbolId: %w", err)
		}
		limit := clampContextMax(req.MaxResult, 10)
		pattern := `\b` + regexp.QuoteMeta(strings.TrimSpace(ref.Name)) + `\b`
		cmd := exec.Command("rg", "-n", pattern, strings.TrimSpace(params.WorkspaceRoot))
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil && stdout.Len() == 0 {
			return out, nil
		}
		sc := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
		count := 0
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			out.Notes = append(out.Notes, "usage: "+line)
			count++
			if count >= limit {
				break
			}
		}
		return out, nil
	default:
		return out, fmt.Errorf("unsupported request_context kind %q", req.Kind)
	}
}

func clampContextMax(v int, def int) int {
	if v <= 0 {
		return def
	}
	if v > 50 {
		return 50
	}
	return v
}

func estimatePlanningContextBytes(c agent.PlanningContext) int {
	total := 0
	for _, sref := range c.Symbols {
		total += len(sref.ID) + len(sref.Name) + len(sref.Path) + len(sref.Kind) + 16
	}
	for _, ex := range c.Excerpts {
		total += len(ex.Path) + len(ex.Content)
	}
	for _, note := range c.Notes {
		total += len(note)
	}
	return total
}

// buildEditApplyParams loads file text on the daemon when activeFile is set.
// Unsaved editor buffers are not visible until workspace indexing supplies them.
func buildEditExecutionContext(params protocol.VoiceTranscriptParams, next intent.NextIntent) (edits.EditExecutionContext, string) {
	active := strings.TrimSpace(params.ActiveFile)
	workspaceRoot := strings.TrimSpace(params.WorkspaceRoot)
	if next.Kind == intent.NextIntentKindEdit && active == "" {
		return edits.EditExecutionContext{}, "activeFile is required when the next action is an edit"
	}
	if next.Kind == intent.NextIntentKindEdit && workspaceRoot == "" {
		return edits.EditExecutionContext{}, "workspaceRoot is required when the next action is an edit"
	}
	fileText := ""
	if active != "" {
		b, err := os.ReadFile(active)
		if err != nil {
			return edits.EditExecutionContext{}, fmt.Sprintf("read active file: %v", err)
		}
		fileText = string(b)
	}
	return edits.EditExecutionContext{
		Instruction:   params.Text,
		ActiveFile:    params.ActiveFile,
		FileText:      fileText,
		WorkspaceRoot: workspaceRoot,
	}, ""
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func toProtocolNavigationIntent(n intent.NavigationIntent) *protocol.NavigationIntent {
	out := &protocol.NavigationIntent{
		Kind: string(n.Kind),
	}
	if n.OpenFile != nil {
		out.OpenFile = &struct {
			Path string `json:"path"`
		}{Path: n.OpenFile.Path}
	}
	if n.RevealSymbol != nil {
		out.RevealSymbol = &struct {
			Path       string `json:"path,omitempty"`
			SymbolName string `json:"symbolName"`
			SymbolKind string `json:"symbolKind,omitempty"`
		}{
			Path:       n.RevealSymbol.Path,
			SymbolName: n.RevealSymbol.SymbolName,
			SymbolKind: n.RevealSymbol.SymbolKind,
		}
	}
	if n.MoveCursor != nil {
		out.MoveCursor = &struct {
			Target struct {
				Path string `json:"path,omitempty"`
				Line int64  `json:"line"`
				Char int64  `json:"char"`
			} `json:"target"`
		}{}
		out.MoveCursor.Target.Path = n.MoveCursor.Target.Path
		out.MoveCursor.Target.Line = int64(n.MoveCursor.Target.Line)
		out.MoveCursor.Target.Char = int64(n.MoveCursor.Target.Char)
	}
	if n.SelectRange != nil {
		out.SelectRange = &struct {
			Target struct {
				Path      string `json:"path,omitempty"`
				StartLine int64  `json:"startLine"`
				StartChar int64  `json:"startChar"`
				EndLine   int64  `json:"endLine"`
				EndChar   int64  `json:"endChar"`
			} `json:"target"`
		}{}
		out.SelectRange.Target.Path = n.SelectRange.Target.Path
		out.SelectRange.Target.StartLine = int64(n.SelectRange.Target.StartLine)
		out.SelectRange.Target.StartChar = int64(n.SelectRange.Target.StartChar)
		out.SelectRange.Target.EndLine = int64(n.SelectRange.Target.EndLine)
		out.SelectRange.Target.EndChar = int64(n.SelectRange.Target.EndChar)
	}
	if n.RevealEdit != nil {
		out.RevealEdit = &struct {
			EditId string `json:"editId"`
		}{EditId: n.RevealEdit.EditID}
	}
	return out
}
