package service

import (
	"context"
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/flows"
	"vocoding.net/vocode/v2/apps/core/internal/flows/router"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func (s *Service) closeSelectionPhase(vs *session.VoiceSession) {
	vs.SearchResults = nil
	vs.ActiveSearchIndex = 0
	vs.PendingDirectiveApply = nil
	vs.BasePhase = session.BasePhaseMain
}

func (s *Service) applySelectionControlOp(vs *session.VoiceSession, op string, pick1Based int) {
	op = strings.ToLower(strings.TrimSpace(op))
	n := len(vs.SearchResults)
	switch op {
	case "next":
		if n > 0 && vs.ActiveSearchIndex < n-1 {
			vs.ActiveSearchIndex++
		}
	case "back":
		if vs.ActiveSearchIndex > 0 {
			vs.ActiveSearchIndex--
		}
	case "pick":
		if pick1Based >= 1 && pick1Based <= n {
			vs.ActiveSearchIndex = pick1Based - 1
		}
	}
}

func (s *Service) selectionApplyHostForActiveHit(params protocol.VoiceTranscriptParams, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, string) {
	if len(vs.SearchResults) == 0 {
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "search results",
			TranscriptOutcome: "selection_control",
			UiDisposition:     "hidden",
			SearchResults:     wireHitsToProtocol(vs.SearchResults),
			ActiveSearchIndex: ptrInt64(int64(vs.ActiveSearchIndex)),
		}, ""
	}
	if s.hostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "daemon has directives but no host apply client is configured"
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
		return protocol.VoiceTranscriptCompletion{Success: false}, "host.applyDirectives failed: " + err.Error()
	}
	if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
		vs.PendingDirectiveApply = nil
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply failed: " + err.Error()
	}
	vs.PendingDirectiveApply = nil
	return protocol.VoiceTranscriptCompletion{
		Success:           true,
		Summary:           "search results",
		TranscriptOutcome: "selection_control",
		UiDisposition:     "hidden",
		SearchResults:     wireHitsToProtocol(vs.SearchResults),
		ActiveSearchIndex: ptrInt64(int64(vs.ActiveSearchIndex)),
	}, ""
}

func (s *Service) dispatchSelectionFlow(
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
) (protocol.VoiceTranscriptCompletion, string) {
	fr, err := s.flowRouter.ClassifyFlow(context.Background(), router.Context{
		Flow:        flows.Select,
		Instruction: text,
		Editor:      editorSnapshotFromParams(params),
		HitCount:    len(vs.SearchResults),
		ActiveIndex: vs.ActiveSearchIndex,
	})
	if err != nil {
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""
	}

	switch fr.Route {
	case "control":
		if !handleGlobalControlRoute(text) {
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				TranscriptOutcome: "irrelevant",
				UiDisposition:     "skipped",
			}, ""
		}
		s.closeSelectionPhase(vs)
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "Search session closed",
			TranscriptOutcome: "selection_control",
			UiDisposition:     "hidden",
		}, ""

	case "select_control":
		op, pick, ok := selectListControlFromText(text)
		if !ok {
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				TranscriptOutcome: "irrelevant",
				UiDisposition:     "skipped",
			}, ""
		}
		s.applySelectionControlOp(vs, op, pick)
		return s.selectionApplyHostForActiveHit(params, vs)

	case "select":
		q := heuristicSearchQuery(text)
		if res, ok, reason := s.searchFromQuery(params, q, vs); ok {
			if strings.TrimSpace(reason) != "" {
				return protocol.VoiceTranscriptCompletion{Success: false}, reason
			}
			return res, ""
		}
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""

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
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""

	case "edit", "delete":
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""

	case "irrelevant":
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			TranscriptOutcome: "irrelevant",
			UiDisposition:     "skipped",
		}, ""

	default:
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""
	}
}

func (s *Service) closeFileSelectionPhase(vs *session.VoiceSession) {
	vs.FileSelectionPaths = nil
	vs.FileSelectionIndex = 0
	vs.FileSelectionFocus = ""
	vs.BasePhase = session.BasePhaseMain
}

func (s *Service) applyFileSelectionControlOp(vs *session.VoiceSession, op string, pick1Based int) {
	op = strings.ToLower(strings.TrimSpace(op))
	n := len(vs.FileSelectionPaths)
	switch op {
	case "next":
		if n > 0 && vs.FileSelectionIndex < n-1 {
			vs.FileSelectionIndex++
		}
	case "back":
		if vs.FileSelectionIndex > 0 {
			vs.FileSelectionIndex--
		}
	case "pick":
		if pick1Based >= 1 && pick1Based <= n {
			vs.FileSelectionIndex = pick1Based - 1
		}
	}
	if n > 0 && vs.FileSelectionIndex >= 0 && vs.FileSelectionIndex < n {
		vs.FileSelectionFocus = vs.FileSelectionPaths[vs.FileSelectionIndex]
	}
}

func (s *Service) fileSelectionOpenPath(params protocol.VoiceTranscriptParams, vs *session.VoiceSession, path string) (protocol.VoiceTranscriptCompletion, string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "open: no file path"
	}
	if s.hostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "daemon has directives but no host apply client is configured"
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
					}{Path: path},
				},
			},
		},
	}
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
		return protocol.VoiceTranscriptCompletion{Success: false}, "host.applyDirectives failed: " + err.Error()
	}
	if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
		vs.PendingDirectiveApply = nil
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply failed: " + err.Error()
	}
	vs.PendingDirectiveApply = nil
	vs.FileSelectionFocus = path
	return protocol.VoiceTranscriptCompletion{
		Success:                true,
		Summary:                "open file",
		TranscriptOutcome:      "file_selection_control",
		UiDisposition:          "hidden",
		FileSelectionFocusPath: path,
	}, ""
}

func (s *Service) fileSelectionDeletePath(params protocol.VoiceTranscriptParams, vs *session.VoiceSession, path string) (protocol.VoiceTranscriptCompletion, string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete: no file path"
	}
	if s.hostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "daemon has directives but no host apply client is configured"
	}
	st, err := os.Stat(path)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete: " + err.Error()
	}
	if st.IsDir() {
		return protocol.VoiceTranscriptCompletion{Success: false}, "delete folder not supported; delete a file"
	}
	dirs := []protocol.VoiceTranscriptDirective{
		{Kind: "delete_file", DeleteFileDirective: &protocol.DeleteFileDirective{Path: path}},
	}
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
		return protocol.VoiceTranscriptCompletion{Success: false}, "host.applyDirectives failed: " + err.Error()
	}
	if err := pending.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
		vs.PendingDirectiveApply = nil
		return protocol.VoiceTranscriptCompletion{Success: false}, "host apply failed: " + err.Error()
	}
	vs.PendingDirectiveApply = nil
	vs.FileSelectionPaths = nil
	return protocol.VoiceTranscriptCompletion{
		Success:           true,
		Summary:           "delete file",
		TranscriptOutcome: "file_selection",
		UiDisposition:     "shown",
	}, ""
}

func (s *Service) dispatchSelectFileFlow(
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
) (protocol.VoiceTranscriptCompletion, string) {
	fr, err := s.flowRouter.ClassifyFlow(context.Background(), router.Context{
		Flow:            flows.SelectFile,
		Instruction:     text,
		Editor:          editorSnapshotFromParams(params),
		FocusPath:       vs.FileSelectionFocus,
		ListCount:       len(vs.FileSelectionPaths),
		ListActiveIndex: vs.FileSelectionIndex,
	})
	if err != nil {
		return protocol.VoiceTranscriptCompletion{}, err.Error()
	}
	resolvedPath := strings.TrimSpace(vs.FileSelectionFocus)

	switch fr.Route {
	case "control":
		if !handleGlobalControlRoute(text) {
			return protocol.VoiceTranscriptCompletion{
				Success:                true,
				TranscriptOutcome:      "irrelevant",
				UiDisposition:          "skipped",
				FileSelectionFocusPath: vs.FileSelectionFocus,
			}, ""
		}
		s.closeFileSelectionPhase(vs)
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "File selection closed",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""

	case "select_file_control":
		op, pick, ok := selectListControlFromText(text)
		if !ok {
			return protocol.VoiceTranscriptCompletion{
				Success:                true,
				TranscriptOutcome:      "irrelevant",
				UiDisposition:          "skipped",
				FileSelectionFocusPath: vs.FileSelectionFocus,
			}, ""
		}
		s.applyFileSelectionControlOp(vs, op, pick)
		return protocol.VoiceTranscriptCompletion{
			Success:                true,
			Summary:                "file focus updated",
			TranscriptOutcome:      "file_selection_control",
			UiDisposition:          "hidden",
			FileSelectionFocusPath: vs.FileSelectionFocus,
		}, ""

	case "select":
		q := heuristicSearchQuery(text)
		if res, ok, reason := s.searchFromQuery(params, q, vs); ok {
			if strings.TrimSpace(reason) != "" {
				return protocol.VoiceTranscriptCompletion{Success: false}, reason
			}
			return res, ""
		}
		return protocol.VoiceTranscriptCompletion{
			Success:                true,
			Summary:                "file focus updated",
			TranscriptOutcome:      "file_selection_control",
			UiDisposition:          "hidden",
			FileSelectionFocusPath: vs.FileSelectionFocus,
		}, ""

	case "select_file":
		q := heuristicSearchQuery(text)
		if res, ok, reason := s.fileSearchFromQuery(params, q, vs); ok {
			if strings.TrimSpace(reason) != "" {
				return protocol.VoiceTranscriptCompletion{Success: false}, reason
			}
			return res, ""
		}
		return protocol.VoiceTranscriptCompletion{
			Success:                true,
			Summary:                "file focus updated",
			TranscriptOutcome:      "file_selection_control",
			UiDisposition:          "hidden",
			FileSelectionFocusPath: vs.FileSelectionFocus,
		}, ""

	case "open":
		return s.fileSelectionOpenPath(params, vs, resolvedPath)
	case "delete":
		return s.fileSelectionDeletePath(params, vs, resolvedPath)
	case "move", "rename", "create":
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			Summary:           "core transcript (stub)",
			TranscriptOutcome: "completed",
			UiDisposition:     "hidden",
		}, ""

	case "irrelevant":
		return protocol.VoiceTranscriptCompletion{
			Success:           true,
			TranscriptOutcome: "irrelevant",
			UiDisposition:     "skipped",
		}, ""

	default:
		return protocol.VoiceTranscriptCompletion{
			Success:                true,
			Summary:                "file focus updated",
			TranscriptOutcome:      "file_selection_control",
			UiDisposition:          "hidden",
			FileSelectionFocusPath: vs.FileSelectionFocus,
		}, ""
	}
}

// fileSelectionLegacyKeywordOps handles open/delete when no model agent is configured.
func (s *Service) fileSelectionLegacyKeywordOps(
	params protocol.VoiceTranscriptParams,
	vs *session.VoiceSession,
	text string,
) (protocol.VoiceTranscriptCompletion, string, bool) {
	tl := strings.ToLower(strings.TrimSpace(text))
	if vs.FileSelectionFocus == "" || s.hostApply == nil {
		return protocol.VoiceTranscriptCompletion{}, "", false
	}
	if strings.Contains(tl, "open") || strings.Contains(tl, "show") || strings.Contains(tl, "reveal") {
		res, fail := s.fileSelectionOpenPath(params, vs, vs.FileSelectionFocus)
		return res, fail, true
	}
	if strings.Contains(tl, "delete") || strings.Contains(tl, "remove") || strings.Contains(tl, "trash") {
		res, fail := s.fileSelectionDeletePath(params, vs, vs.FileSelectionFocus)
		return res, fail, true
	}
	return protocol.VoiceTranscriptCompletion{}, "", false
}
