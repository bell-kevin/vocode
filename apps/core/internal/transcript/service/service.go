package service

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	global "vocoding.net/vocode/v2/apps/core/internal/flows/global"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/flows/selection"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/clarify"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// Service is the core transcript controller: manages session state and flow/clarify lifecycle.
// For now, it stubs out the executor and returns "completed" while laying down the
// session/phase model required for the real port.
type Service struct {
	sessions  *session.VoiceSessionStore
	ephemeral session.VoiceSession

	hostApply  hostApplyClient
	flowRouter *router.FlowRouter
}

func NewService(flowRouter *router.FlowRouter) *Service {
	if flowRouter == nil {
		flowRouter = router.NewFlowRouter(nil)
	}
	return &Service{
		sessions:   session.NewVoiceSessionStore(),
		flowRouter: flowRouter,
	}
}

type hostApplyClient interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}

func (s *Service) SetHostApplyClient(client hostApplyClient) {
	s.hostApply = client
}

func (s *Service) AcceptTranscript(
	params protocol.VoiceTranscriptParams,
) (protocol.VoiceTranscriptCompletion, bool, string) {
	key := strings.TrimSpace(params.ContextSessionId)
	var vs session.VoiceSession
	if key == "" {
		vs = session.CloneVoiceSession(s.ephemeral)
	} else {
		// idle eviction and exact gathered caps are ported later; for now idleReset is 0.
		vs = s.sessions.Get(key, s.idleReset(params))
	}

	// Handle control requests first (no spoken text required).
	if cr := strings.TrimSpace(params.ControlRequest); cr != "" {
		out, ok := s.handleControlRequest(params, key, &vs, cr)
		s.persist(key, vs)
		return out, ok, ""
	}

	text := strings.TrimSpace(params.Text)
	if text == "" {
		// Match daemon semantics: empty transcript is invalid params (handler returns ok=false).
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}

	// Clarify overlay handling: answer and dismiss, or exit phrases -> clarify_control.
	if vs.Clarify != nil {
		if global.IsExitPhrase(text) {
			vs.Clarify = nil
			s.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Clarification cancelled",
				TranscriptOutcome: "clarify_control",
				UiDisposition:     "hidden",
			}, true, ""
		}

		// Treat as answer.
		ans := text
		ov := *vs.Clarify

		// Resume: execute whatever clarifyTargetResolution would have done (flow switching for now).
		// resumeFromClarification reads `vs.Clarify`, so dismiss the overlay after resume policy.
		cc := clarify.BuildClarificationContext(ov, ans)
		_ = cc
		s.resumeFromClarification(&vs)
		vs.Clarify = nil
		s.persist(key, vs)
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "Clarification resolved",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, true, ""
	}

	if vs.BasePhase == "" {
		vs.BasePhase = session.BasePhaseMain
	}

	// Base-phase flow handling: selection navigation exit, file_selection exit.
	switch vs.BasePhase {
	case session.BasePhaseSelection:
		if global.IsExitPhrase(text) {
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.PendingDirectiveApply = nil
			vs.BasePhase = session.BasePhaseMain
			s.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Search session closed",
				TranscriptOutcome: "selection_control",
				UiDisposition:     "hidden",
			}, true, ""
		}

		// selection flow control utterances: update nav index and apply host navigation directives.
		if len(vs.SearchResults) > 0 {
			if navKind, ord, ok := selection.ParseNav(text); ok && navKind != "exit" {
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
					// ord is 1-based.
					if ord >= 1 && ord <= len(vs.SearchResults) {
						vs.ActiveSearchIndex = ord - 1
					}
				}
				if s.hostApply == nil {
					// Keep session change, but surface as a failure since the UX depends on host navigation.
					s.persist(key, vs)
					return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
				}

				hit := vs.SearchResults[vs.ActiveSearchIndex]
				dirs := hitNavigateDirectives(hit.Path, hit.Line, hit.Character, 1)
				batchID := newApplyBatchID()
				pending := &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dirs)}
				vs.PendingDirectiveApply = pending
				hostRes, err := s.hostApply.ApplyDirectives(protocol.HostApplyParams{
					ApplyBatchId: batchID,
					ActiveFile:   params.ActiveFile,
					Directives:   dirs,
				})
				if err != nil {
					vs.PendingDirectiveApply = nil
					s.persist(key, vs)
					return protocol.VoiceTranscriptCompletion{Success: false}, true, "host.applyDirectives failed: " + err.Error()
				}
				if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
					vs.PendingDirectiveApply = nil
					s.persist(key, vs)
					return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply failed: " + err.Error()
				}
				vs.PendingDirectiveApply = nil

				s.persist(key, vs)
				return protocol.VoiceTranscriptCompletion{
					Success:           true,
					Summary:           "search results",
					TranscriptOutcome: "selection_control",
					UiDisposition:     "hidden",
					SearchResults:     wireHitsToProtocol(vs.SearchResults),
					ActiveSearchIndex: ptrInt64(int64(vs.ActiveSearchIndex)),
				}, true, ""
			}
		}

		execRes, failure := s.dispatchSelectionFlow(params, &vs, text)
		if strings.TrimSpace(failure) != "" {
			s.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{Success: false}, true, failure
		}
		s.applyTranscriptOutcome(&vs, params, execRes)
		s.persist(key, vs)
		return execRes, true, ""

	case session.BasePhaseFileSelection:
		if global.IsExitPhrase(text) {
			vs.FileSelectionPaths = nil
			vs.FileSelectionIndex = 0
			vs.FileSelectionFocus = ""
			vs.BasePhase = session.BasePhaseMain
			s.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "File selection closed",
				TranscriptOutcome: "completed",
				UiDisposition:     "hidden",
			}, true, ""
		}

		// File selection is always a search hit list (paths); there is no whole-workspace browse.
		if len(vs.FileSelectionPaths) == 0 {
			vs.BasePhase = session.BasePhaseMain
			s.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "No file hits in this session; use find file … (or your assistant’s file search) to get a path list first.",
				TranscriptOutcome: "completed",
				UiDisposition:     "shown",
			}, true, ""
		}

		if focused := strings.TrimSpace(params.FocusedWorkspacePath); focused != "" {
			for i, p := range vs.FileSelectionPaths {
				if p == focused {
					vs.FileSelectionIndex = i
					vs.FileSelectionFocus = p
					break
				}
			}
		}

		tl := strings.ToLower(strings.TrimSpace(text))

		if s.flowRouter.Model == nil {
			if res, fail, ok := s.fileSelectionLegacyKeywordOps(params, &vs, text); ok {
				if strings.TrimSpace(fail) != "" {
					s.persist(key, vs)
					return protocol.VoiceTranscriptCompletion{Success: false}, true, fail
				}
				s.persist(key, vs)
				return res, true, ""
			}
			if (strings.Contains(tl, "open") || strings.Contains(tl, "delete") || strings.Contains(tl, "remove")) && s.hostApply == nil {
				s.persist(key, vs)
				return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
			}
		}

		navKind, ord, okNav := selection.ParseNav(text)
		parsedNav := okNav && navKind != "exit" && len(vs.FileSelectionPaths) > 0
		if parsedNav {
			switch navKind {
			case "next":
				if vs.FileSelectionIndex < len(vs.FileSelectionPaths)-1 {
					vs.FileSelectionIndex++
				}
			case "back":
				if vs.FileSelectionIndex > 0 {
					vs.FileSelectionIndex--
				}
			case "pick":
				if ord >= 1 && ord <= len(vs.FileSelectionPaths) {
					vs.FileSelectionIndex = ord - 1
				}
			}
			vs.FileSelectionFocus = vs.FileSelectionPaths[vs.FileSelectionIndex]
		}

		if parsedNav {
			s.persist(key, vs)
			return protocol.VoiceTranscriptCompletion{
				Success:                true,
				Summary:                "file focus updated",
				TranscriptOutcome:      "file_selection_control",
				UiDisposition:          "hidden",
				FileSelectionFocusPath: vs.FileSelectionFocus,
			}, true, ""
		}

		frRes, frFail := s.dispatchSelectFileFlow(params, &vs, text)
		if strings.TrimSpace(frFail) != "" {
			s.persist(key, vs)
			if !frRes.Success {
				return frRes, true, frFail
			}
			return protocol.VoiceTranscriptCompletion{Success: false}, true, frFail
		}
		s.persist(key, vs)
		s.applyTranscriptOutcome(&vs, params, frRes)
		return frRes, true, ""
	}

	// Main/default: stub completion.
	execRes, failure := s.executeStub(params, &vs)
	if strings.TrimSpace(failure) != "" {
		s.persist(key, vs)
		return protocol.VoiceTranscriptCompletion{Success: false}, true, failure
	}
	s.applyTranscriptOutcome(&vs, params, execRes)
	s.persist(key, vs)
	return execRes, true, ""
}

func (s *Service) idleReset(params protocol.VoiceTranscriptParams) time.Duration {
	if params.DaemonConfig == nil || params.DaemonConfig.SessionIdleResetMs == nil {
		return 0
	}
	ms := *params.DaemonConfig.SessionIdleResetMs
	if ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func (s *Service) persist(key string, vs session.VoiceSession) {
	if strings.TrimSpace(key) == "" {
		s.ephemeral = session.CloneVoiceSession(vs)
		return
	}
	s.sessions.Put(key, vs)
}

func ptrInt64(v int64) *int64 { return &v }

func editorSnapshotFromParams(params protocol.VoiceTranscriptParams) router.EditorSnapshot {
	return router.EditorSnapshot{
		ActiveFilePath: strings.TrimSpace(params.ActiveFile),
		WorkspaceRoot:  strings.TrimSpace(params.WorkspaceRoot),
	}
}

func wireHitsToProtocol(in []session.SearchHit) []struct {
	Path      string `json:"path"`
	Line      int64  `json:"line"`
	Character int64  `json:"character"`
	Preview   string `json:"preview"`
} {
	out := make([]struct {
		Path      string `json:"path"`
		Line      int64  `json:"line"`
		Character int64  `json:"character"`
		Preview   string `json:"preview"`
	}, 0, len(in))
	for _, h := range in {
		out = append(out, struct {
			Path      string `json:"path"`
			Line      int64  `json:"line"`
			Character int64  `json:"character"`
			Preview   string `json:"preview"`
		}{
			Path:      h.Path,
			Line:      int64(h.Line),
			Character: int64(h.Character),
			Preview:   h.Preview,
		})
	}
	return out
}

func sessionHitsFromProtocol(in []struct {
	Path      string `json:"path"`
	Line      int64  `json:"line"`
	Character int64  `json:"character"`
	Preview   string `json:"preview"`
},
) []session.SearchHit {
	out := make([]session.SearchHit, 0, len(in))
	for _, h := range in {
		out = append(out, session.SearchHit{
			Path:      h.Path,
			Line:      int(h.Line),
			Character: int(h.Character),
			Preview:   h.Preview,
		})
	}
	return out
}

// executeStub is a placeholder until capabilities are ported.
func (s *Service) executeStub(
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
) (protocol.VoiceTranscriptCompletion, string) {
	text := strings.TrimSpace(params.Text)
	if text == "" {
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""
	}

	fr, err := s.flowRouter.ClassifyFlow(context.Background(), router.Context{
		Flow:        flows.Root,
		Instruction: text,
		Editor:      editorSnapshotFromParams(params),
	})
	if err == nil {
		switch fr.Route {
		case "select":
			q := heuristicSearchQuery(text)
			if res, ok, reason := s.searchFromQuery(params, q, vs); ok {
				if strings.TrimSpace(reason) != "" {
					return protocol.VoiceTranscriptCompletion{Success: false}, reason
				}
				return res, ""
			}
		case "select_file":
			q := heuristicSearchQuery(text)
			if res, ok, reason := s.fileSearchFromQuery(params, q, vs); ok {
				if strings.TrimSpace(reason) != "" {
					return protocol.VoiceTranscriptCompletion{Success: false}, reason
				}
				return res, ""
			}
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				TranscriptOutcome: "irrelevant",
				UiDisposition:     "skipped",
			}, ""
		case "question":
			ans := stubQuestionAnswer()
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				TranscriptOutcome: "answer",
				UiDisposition:     "hidden",
				AnswerText:        ans,
				Summary:           ans,
			}, ""
		case "irrelevant":
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				TranscriptOutcome: "irrelevant",
				UiDisposition:     "skipped",
			}, ""
		case "control":
			if global.IsExitPhrase(text) {
				return protocol.VoiceTranscriptCompletion{
					Success:           true,
					Summary:           "core transcript (stub)",
					TranscriptOutcome: "completed",
					UiDisposition:     "hidden",
				}, ""
			}
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				TranscriptOutcome: "irrelevant",
				UiDisposition:     "skipped",
			}, ""
		}
	}

	if res, ok, reason := s.searchFromText(params, text, vs); ok {
		if strings.TrimSpace(reason) != "" {
			return protocol.VoiceTranscriptCompletion{Success: false}, reason
		}
		return res, ""
	}

	if q, ok := FileSearchLikeQueryFromText(text); ok {
		if res, ok2, reason := s.fileSearchFromQuery(params, q, vs); ok2 {
			if strings.TrimSpace(reason) != "" {
				return protocol.VoiceTranscriptCompletion{Success: false}, reason
			}
			return res, ""
		}
	}

	return protocol.VoiceTranscriptCompletion{
		Success:           true,
		TranscriptOutcome: "irrelevant",
		UiDisposition:     "skipped",
	}, ""
}

var applyBatchSeq uint64

func newApplyBatchID() string {
	seq := atomic.AddUint64(&applyBatchSeq, 1)
	return fmt.Sprintf("core-%d", seq)
}

func hitNavigateDirectives(path string, line0, char0, length int) []protocol.VoiceTranscriptDirective {
	open := protocol.VoiceTranscriptDirective{
		Kind: "navigate",
		NavigationDirective: &protocol.NavigationDirective{
			Kind: "success",
			Action: &protocol.NavigationAction{
				Kind: "open_file",
				OpenFile: &struct {
					Path string `json:"path"`
				}{Path: path},
			},
		},
	}
	sel := protocol.VoiceTranscriptDirective{
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
						Path:      path,
						StartLine: int64(line0),
						StartChar: int64(char0),
						EndLine:   int64(line0),
						EndChar:   int64(char0 + length),
					},
				},
			},
		},
	}
	return []protocol.VoiceTranscriptDirective{open, sel}
}

// applyTranscriptOutcome mutates session state based on transcript outcome.
// This is where selection/file_selection/clarify flow transitions are centralized.
func (s *Service) applyTranscriptOutcome(
	vs *session.VoiceSession,
	params protocol.VoiceTranscriptParams,
	res protocol.VoiceTranscriptCompletion,
) {
	if vs == nil {
		return
	}

	switch res.TranscriptOutcome {
	case "search", "selection", "selection_control":
		if len(res.SearchResults) > 0 {
			vs.SearchResults = sessionHitsFromProtocol(res.SearchResults)
			if res.ActiveSearchIndex != nil {
				i := int(*res.ActiveSearchIndex)
				if i < 0 {
					i = 0
				}
				if i >= len(vs.SearchResults) {
					i = 0
				}
				vs.ActiveSearchIndex = i
			} else {
				vs.ActiveSearchIndex = 0
			}
			vs.BasePhase = session.BasePhaseSelection
		} else if res.SearchResults != nil {
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.BasePhase = session.BasePhaseMain
		}

	case "file_selection", "file_selection_control":
		vs.BasePhase = session.BasePhaseFileSelection
		if res.FileSelectionFocusPath != "" {
			vs.FileSelectionFocus = res.FileSelectionFocusPath
		}
		// Switching into file_selection exits search-only state.
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0

	case "clarify":
		if strings.TrimSpace(res.Summary) == "" {
			return
		}
		if err := clarify.ValidateForBasePhase(vs.BasePhase, res.ClarifyTargetResolution); err != nil {
			// Leave session unchanged on invalid clarify target.
			return
		}
		vs.Clarify = &session.ClarifyOverlay{
			TargetResolution:   res.ClarifyTargetResolution,
			Question:           strings.TrimSpace(res.Summary),
			OriginalTranscript: strings.TrimSpace(params.Text),
		}
	}
}

func (s *Service) handleControlRequest(
	params protocol.VoiceTranscriptParams,
	key string,
	vs *session.VoiceSession,
	cr string,
) (protocol.VoiceTranscriptCompletion, bool) {
	_ = params

	switch cr {
	case "cancel_clarify":
		if vs.Clarify != nil {
			vs.Clarify = nil
		}
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "Clarification cancelled",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, true

	case "cancel_selection":
		// cancel_selection closes the selection base phase and any clarify overlay.
		vs.Clarify = nil
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0
		vs.PendingDirectiveApply = nil
		if vs.BasePhase == session.BasePhaseSelection {
			vs.BasePhase = session.BasePhaseMain
		}
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "Search session closed",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, true
	default:
		return protocol.VoiceTranscriptCompletion{}, false
	}
}

func (s *Service) resumeFromClarification(vs *session.VoiceSession) {
	// Stub resume policy aligned with the plan:
	// - based on clarifications.targetResolution and current base phase, switch flows,
	// - close selection after edit while in selection (selection lifecycle policy: any-selection-edit).
	//
	// Once the executor is ported, this will be replaced by "execute original pipeline branch
	// for the target, but with structured ClarificationContext".
	//
	// For now, infer using the stored overlay target resolution string.
	if vs.Clarify == nil {
		return
	}

	switch vs.Clarify.TargetResolution {
	case clarify.ClarifyTargetSelect:
		vs.BasePhase = session.BasePhaseSelection
		// Switching flows exits the other flow’s state.
		vs.FileSelectionPaths = nil
		vs.FileSelectionIndex = 0
		vs.FileSelectionFocus = ""
	case clarify.ClarifyTargetSelectFile:
		vs.BasePhase = session.BasePhaseFileSelection
		vs.SearchResults = nil
		vs.ActiveSearchIndex = 0
		vs.PendingDirectiveApply = nil
	case clarify.ClarifyTargetEdit:
		if vs.BasePhase == session.BasePhaseSelection {
			vs.BasePhase = session.BasePhaseMain
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.PendingDirectiveApply = nil
		}
	default:
		// Keep main.
	}
}
