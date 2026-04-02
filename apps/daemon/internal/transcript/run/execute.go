package run

import (
	"strings"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/config"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/executor"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func Execute(env *Env, params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string) {
	key := strings.TrimSpace(params.ContextSessionId)
	dc := params.DaemonConfig

	idleReset := config.DefaultSessionIdleReset()
	if dc != nil && dc.SessionIdleResetMs != nil {
		ms := *dc.SessionIdleResetMs
		if ms == 0 {
			idleReset = 0
		} else if ms > 0 {
			idleReset = time.Duration(ms) * time.Millisecond
		} else {
			idleReset = config.DefaultSessionIdleReset()
		}
	}
	var vs agentcontext.VoiceSession
	if strings.TrimSpace(key) == "" {
		vs = voicesession.Load(env.Sessions, key, idleReset, env.Ephemeral)
	} else {
		vs = voicesession.Load(env.Sessions, key, idleReset, nil)
	}
	syncSelectionStackForHits(&vs)

	if cr := strings.TrimSpace(params.ControlRequest); cr != "" {
		switch cr {
		case "cancel_selection":
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.PendingDirectiveApply = nil
			vs.FlowStack = agentcontext.PopWhileTopKind(vs.FlowStack, agentcontext.FlowKindClarify, agentcontext.FlowKindSelection)
			env.persistSession(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Search session closed",
				TranscriptOutcome: "completed",
				UiDisposition:     "hidden",
			}, true, ""
		case "cancel_clarify":
			if ns, ok := agentcontext.FlowPopIfTop(vs.FlowStack, agentcontext.FlowKindClarify); ok {
				vs.FlowStack = ns
			}
			env.persistSession(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Clarification cancelled",
				TranscriptOutcome: "completed",
				UiDisposition:     "hidden",
			}, true, ""
		default:
			return protocol.VoiceTranscriptCompletion{Success: false}, false, "unknown controlRequest"
		}
	}

	inClarify := agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindClarify
	if inClarify {
		t := strings.TrimSpace(strings.ToLower(params.Text))
		if t != "" && clarifyControlExitRe.MatchString(t) {
			if ns, ok := agentcontext.FlowPopIfTop(vs.FlowStack, agentcontext.FlowKindClarify); ok {
				vs.FlowStack = ns
			}
			env.persistSession(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Clarification cancelled",
				TranscriptOutcome: "clarify_control",
				UiDisposition:     "hidden",
			}, true, ""
		}

		answer := strings.TrimSpace(params.Text)
		if answer != "" {
			q, orig, _, _ := agentcontext.ClarifyPromptFromStack(vs.FlowStack)
			params.Text = strings.Join([]string{
				orig,
				"",
				"Clarifying question: " + q,
				"User answer: " + answer,
			}, "\n")
		}
		if ns, ok := agentcontext.FlowPopIfTop(vs.FlowStack, agentcontext.FlowKindClarify); ok {
			vs.FlowStack = ns
		}
	}

	if agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindFileSelection {
		t := strings.TrimSpace(strings.ToLower(params.Text))
		if t != "" && fileSelectionExitRe.MatchString(t) {
			if ns, ok := agentcontext.FlowPopIfTop(vs.FlowStack, agentcontext.FlowKindFileSelection); ok {
				vs.FlowStack = ns
			}
			vs.FileSelectionPaths = nil
			vs.FileSelectionIndex = 0
			vs.FileSelectionFocus = ""
			env.persistSession(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "File selection closed",
				TranscriptOutcome: "completed",
				UiDisposition:     "hidden",
			}, true, ""
		}
	}

	maxGatheredBytes := config.DefaultGatheredMaxBytes
	maxGatheredExcerpts := config.DefaultGatheredMaxExcerpts
	if dc != nil {
		if dc.MaxGatheredBytes != nil {
			maxGatheredBytes = int(*dc.MaxGatheredBytes)
		}
		if dc.MaxGatheredExcerpts != nil {
			maxGatheredExcerpts = int(*dc.MaxGatheredExcerpts)
		}
	}

	if agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindSelection &&
		len(vs.SearchResults) > 0 {
		if navKind, ord, ok := parseSelectionNav(params.Text); ok {
			if navKind == "exit" {
				vs.SearchResults = nil
				vs.ActiveSearchIndex = 0
				vs.PendingDirectiveApply = nil
				vs.FlowStack = agentcontext.PopWhileTopKind(vs.FlowStack, agentcontext.FlowKindClarify, agentcontext.FlowKindSelection)
				env.persistSession(key, vs)
				return protocol.VoiceTranscriptCompletion{
					Success:           true,
					Summary:           "Search session closed",
					TranscriptOutcome: "selection_control",
					UiDisposition:     "hidden",
				}, true, ""
			}
			switch navKind {
			case "next":
				if vs.ActiveSearchIndex < len(vs.SearchResults)-1 {
					vs.ActiveSearchIndex++
				}
			case "back":
				if vs.ActiveSearchIndex > 0 {
					vs.ActiveSearchIndex--
				}
			case "pick":
				if ord >= 1 && ord <= len(vs.SearchResults) {
					vs.ActiveSearchIndex = ord - 1
				}
			}
			hit := vs.SearchResults[vs.ActiveSearchIndex]
			res := protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "search results",
				TranscriptOutcome: "selection_control",
				UiDisposition:     "hidden",
				SearchResults:     voiceSessionHitsToWire(vs.SearchResults),
				ActiveSearchIndex: ptrInt64(int64(vs.ActiveSearchIndex)),
			}
			dirs := []protocol.VoiceTranscriptDirective{
				{
					Kind: "navigate",
					NavigationDirective: &protocol.NavigationDirective{
						Kind: "success",
						Action: &protocol.NavigationAction{
							Kind: "open_file",
							OpenFile: &struct {
								Path string `json:"path"`
							}{Path: hit.Path},
						},
					},
				},
				{
					Kind: "navigate",
					NavigationDirective: &protocol.NavigationDirective{
						Kind: "success",
						Action: &protocol.NavigationAction{
							Kind: "select_range",
							SelectRange: &struct {
								Target struct {
									Path      string `json:"path,omitempty"`
									StartLine int64  `json:"startLine"`
									StartChar int64  `json:"startChar"`
									EndLine   int64  `json:"endLine"`
									EndChar   int64  `json:"endChar"`
								} `json:"target"`
							}{
								Target: struct {
									Path      string `json:"path,omitempty"`
									StartLine int64  `json:"startLine"`
									StartChar int64  `json:"startChar"`
									EndLine   int64  `json:"endLine"`
									EndChar   int64  `json:"endChar"`
								}{
									Path:      hit.Path,
									StartLine: int64(hit.Line),
									StartChar: int64(hit.Character),
									EndLine:   int64(hit.Line),
									EndChar:   int64(hit.Character + 1),
								},
							},
						},
					},
				},
			}
			if env.HostApply == nil {
				return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
			}
			pending := &agentcontext.DirectiveApplyBatch{ID: "search-nav", NumDirectives: len(dirs)}
			vs.PendingDirectiveApply = pending
			hostRes, err := env.HostApply.ApplyDirectives(protocol.HostApplyParams{
				ApplyBatchId: pending.ID,
				ActiveFile:   params.ActiveFile,
				Directives:   dirs,
			})
			_ = hostRes
			_ = err
			vs.PendingDirectiveApply = nil
			env.persistSession(key, vs)
			return res, true, ""
		}
	}

	if agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindFileSelection &&
		strings.TrimSpace(params.Text) != "" {
		res, dirs, reason := HandleFileSelectionUtterance(params, &vs)
		if reason != "" {
			env.persistSession(key, vs)
			return protocol.VoiceTranscriptCompletion{Success: false}, true, reason
		}
		if len(dirs) == 0 {
			applyTranscriptUIDisposition(
				&res,
				agentcontext.FlowTopKind(vs.FlowStack),
				len(vs.SearchResults) > 0,
			)
			if res.TranscriptOutcome == "clarify" && strings.TrimSpace(res.Summary) != "" {
				parentFlow := agentcontext.FlowTopKind(vs.FlowStack)
				target := strings.TrimSpace(res.ClarifyTargetResolution)
				if err := agentcontext.ValidateClarifyTargetResolution(parentFlow, target); err != nil {
					env.persistSession(key, vs)
					return protocol.VoiceTranscriptCompletion{Success: false}, true, err.Error()
				}
				if ns, ok := agentcontext.FlowPush(vs.FlowStack, agentcontext.FlowFrame{
					Kind:                      agentcontext.FlowKindClarify,
					ClarifyTargetResolution:   target,
					ClarifyQuestion:           strings.TrimSpace(res.Summary),
					ClarifyOriginalTranscript: strings.TrimSpace(params.Text),
				}); ok {
					vs.FlowStack = ns
				}
			}
			env.persistSession(key, vs)
			return res, true, ""
		}
		if env.HostApply == nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
		}
		batchID, err := executor.NewDirectiveApplyBatchID()
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, true, err.Error()
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dirs)}
		vs.PendingDirectiveApply = pending
		hostRes, err := env.HostApply.ApplyDirectives(protocol.HostApplyParams{
			ApplyBatchId: pending.ID,
			ActiveFile:   params.ActiveFile,
			Directives:   dirs,
		})
		_ = hostRes
		_ = err
		vs.PendingDirectiveApply = nil
		applyTranscriptUIDisposition(
			&res,
			agentcontext.FlowTopKind(vs.FlowStack),
			len(vs.SearchResults) > 0,
		)
		env.persistSession(key, vs)
		return res, true, ""
	}

	execParams := params
	execOpt := executor.ExecuteOptions{}
	if agentcontext.FlowTopKind(vs.FlowStack) == agentcontext.FlowKindSelection &&
		len(vs.SearchResults) > 0 {
		if _, _, okNav := parseSelectionNav(params.Text); !okNav && strings.TrimSpace(params.Text) != "" {
			hit := vs.SearchResults[vs.ActiveSearchIndex]
			execParams = params
			execParams.ActiveFile = hit.Path
			execParams.CursorPosition = &struct {
				Line      int64 `json:"line"`
				Character int64 `json:"character"`
			}{Line: int64(hit.Line), Character: int64(hit.Character)}
			execParams.ActiveSelection = &struct {
				StartLine int64 `json:"startLine"`
				StartChar int64 `json:"startChar"`
				EndLine   int64 `json:"endLine"`
				EndChar   int64 `json:"endChar"`
			}{
				StartLine: int64(hit.Line),
				StartChar: int64(hit.Character),
				EndLine:   int64(hit.Line),
				EndChar:   int64(hit.Character + 1),
			}
			execParams.ActiveFileSymbols = nil
			execOpt.Mode = executor.FlowModeSelection
		}
	}

	activeFile := strings.TrimSpace(execParams.ActiveFile)

	vs.Gathered = agentcontext.ApplyGatheredRollingCap(
		vs.Gathered,
		activeFile,
		maxGatheredBytes,
		maxGatheredExcerpts,
	)

	res, dirs, g1, pending, ok, reason := env.Executor.Execute(execParams, vs.Gathered, execOpt)
	vs.Gathered = g1
	if strings.TrimSpace(key) != "" {
		vs.Gathered = agentcontext.ApplyGatheredRollingCap(vs.Gathered, activeFile, maxGatheredBytes, maxGatheredExcerpts)
	}

	if !ok {
		env.persistSession(key, vs)
		if strings.TrimSpace(reason) == "" {
			reason = "executor rejected transcript params"
		}
		return protocol.VoiceTranscriptCompletion{Success: false}, false, reason
	}

	applyTranscriptUIDisposition(
		&res,
		agentcontext.FlowTopKind(vs.FlowStack),
		len(vs.SearchResults) > 0,
	)

	if !res.Success || len(dirs) == 0 {
		vs.PendingDirectiveApply = nil
		env.persistSession(key, vs)
		if !res.Success && strings.TrimSpace(reason) != "" {
			return res, ok, reason
		}
		if res.TranscriptOutcome == "search" || res.TranscriptOutcome == "selection" {
			if len(res.SearchResults) > 0 {
				vs.SearchResults = wireHitsToVoiceSession(res.SearchResults)
				if res.ActiveSearchIndex != nil {
					vs.ActiveSearchIndex = int(*res.ActiveSearchIndex)
				} else {
					vs.ActiveSearchIndex = 0
				}
			} else {
				vs.SearchResults = nil
				vs.ActiveSearchIndex = 0
			}
			syncSelectionStackForHits(&vs)
			env.persistSession(key, vs)
		}
		if res.TranscriptOutcome == "file_selection" && agentcontext.FlowTopKind(vs.FlowStack) != agentcontext.FlowKindFileSelection {
			vs.FileSelectionPaths = nil
			vs.FileSelectionIndex = 0
			vs.FileSelectionFocus = ""
			if ns, ok := agentcontext.FlowPush(vs.FlowStack, agentcontext.FlowFrame{Kind: agentcontext.FlowKindFileSelection}); ok {
				vs.FlowStack = ns
			}
			env.persistSession(key, vs)
		}
		if res.TranscriptOutcome == "clarify" && strings.TrimSpace(res.Summary) != "" {
			parentFlow := agentcontext.FlowTopKind(vs.FlowStack)
			target := strings.TrimSpace(res.ClarifyTargetResolution)
			if err := agentcontext.ValidateClarifyTargetResolution(parentFlow, target); err != nil {
				env.persistSession(key, vs)
				return protocol.VoiceTranscriptCompletion{Success: false}, ok, err.Error()
			}
			if ns, ok := agentcontext.FlowPush(vs.FlowStack, agentcontext.FlowFrame{
				Kind:                      agentcontext.FlowKindClarify,
				ClarifyTargetResolution:   target,
				ClarifyQuestion:           strings.TrimSpace(res.Summary),
				ClarifyOriginalTranscript: strings.TrimSpace(execParams.Text),
			}); ok {
				vs.FlowStack = ns
			}
			env.persistSession(key, vs)
		}
		return res, ok, ""
	}

	if pending == nil || env.HostApply == nil {
		vs.PendingDirectiveApply = nil
		env.persistSession(key, vs)
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
	}

	vs.PendingDirectiveApply = pending
	hostRes, err := env.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: pending.ID,
		ActiveFile:   execParams.ActiveFile,
		Directives:   dirs,
	})
	if err != nil {
		vs.PendingDirectiveApply = nil
		env.persistSession(key, vs)
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "host.applyDirectives failed: " + err.Error()
	}
	if err := voicesession.ConsumeHostApplyReport(pending.ID, hostRes.Items, &vs); err != nil {
		vs.PendingDirectiveApply = nil
		env.persistSession(key, vs)
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply failed: " + err.Error()
	}

	if res.TranscriptOutcome == "search" || res.TranscriptOutcome == "selection" {
		if len(res.SearchResults) > 0 {
			vs.SearchResults = wireHitsToVoiceSession(res.SearchResults)
			if res.ActiveSearchIndex != nil {
				vs.ActiveSearchIndex = int(*res.ActiveSearchIndex)
			} else {
				vs.ActiveSearchIndex = 0
			}
		} else {
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
		}
		syncSelectionStackForHits(&vs)
	}

	env.persistSession(key, vs)
	return res, ok, ""
}
