package transcript

import (
	"regexp"
	"strings"
	"time"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/config"
	"vocoding.net/vocode/v2/apps/daemon/internal/transcript/voicesession"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

var (
	searchControlNextRe   = regexp.MustCompile(`\b(next|forward)\b`)
	searchControlBackRe   = regexp.MustCompile(`\b(back|prev|previous)\b`)
	searchControlExitRe   = regexp.MustCompile(`\b(cancel|exit|close|stop|done|quit|leave|end|abort)\b`)
	searchControlResultRe = regexp.MustCompile(`\bresult\b`)
	searchControlEditRe   = regexp.MustCompile(`\bedit\b`)
	searchControlSelectRe = regexp.MustCompile(`\b(select|choose|pick)\b`)
	searchControlGoRe     = regexp.MustCompile(`\b(go|jump|open|show)\b`)
	searchControlIntRe    = regexp.MustCompile(`\b\d+\b`)

	clarifyControlExitRe = regexp.MustCompile(`\b(cancel|exit|close|stop|done|quit|leave|end|abort)\b`)
)

func parseSearchControl(text string) (kind string, ordinal int, ok bool) {
	t := strings.TrimSpace(strings.ToLower(text))
	if t == "" {
		return "", 0, false
	}
	if searchControlExitRe.MatchString(t) {
		return "exit", 0, true
	}
	if searchControlNextRe.MatchString(t) {
		return "next", 0, true
	}
	if searchControlBackRe.MatchString(t) {
		return "back", 0, true
	}

	// "edit result N" and "select result N" should both behave like "result N" (jump+select).
	hasResult := searchControlResultRe.MatchString(t)
	hasEdit := searchControlEditRe.MatchString(t)
	hasSelect := searchControlSelectRe.MatchString(t)
	hasGo := searchControlGoRe.MatchString(t)

	// Bare ordinal only if the whole utterance is essentially "3" / "three" / "third".
	if isBareOrdinal(t) {
		if n := parseAnyOrdinal(t); n > 0 {
			return "pick", n, true
		}
	}

	// Otherwise, require a hint word so random numbers in normal instructions don't hijack.
	// Examples:
	// - "result 3"
	// - "select result three"
	// - "edit result 4"
	// - "go to result 2"
	if hasResult || hasEdit || hasSelect || hasGo {
		if n := parseAnyOrdinal(t); n > 0 {
			return "pick", n, true
		}
	}

	return "", 0, false
}

func parseAnyOrdinal(s string) int {
	if n := parseAnyIntToken(s); n > 0 {
		return n
	}
	for _, w := range strings.Fields(s) {
		switch strings.Trim(w, ".,;:!?") {
		case "one", "1st", "first":
			return 1
		case "two", "2nd", "second":
			return 2
		case "three", "3rd", "third":
			return 3
		case "four", "4th", "fourth":
			return 4
		case "five", "5th", "fifth":
			return 5
		case "six", "6th", "sixth":
			return 6
		case "seven", "7th", "seventh":
			return 7
		case "eight", "8th", "eighth":
			return 8
		case "nine", "9th", "ninth":
			return 9
		case "ten", "10th", "tenth":
			return 10
		}
	}
	return 0
}

func isBareOrdinal(t string) bool {
	t = strings.TrimSpace(strings.ToLower(t))
	if t == "" {
		return false
	}
	// Digits only (optionally surrounded by punctuation/whitespace).
	if searchControlIntRe.MatchString(t) && len(strings.Fields(t)) == 1 {
		return true
	}
	switch strings.Trim(t, ".,;:!?") {
	case "one", "first", "1st",
		"two", "second", "2nd",
		"three", "third", "3rd",
		"four", "fourth", "4th",
		"five", "fifth", "5th",
		"six", "sixth", "6th",
		"seven", "seventh", "7th",
		"eight", "eighth", "8th",
		"nine", "ninth", "9th",
		"ten", "tenth", "10th":
		return true
	}
	return false
}

func parseAnyIntToken(s string) int {
	n := 0
	inDigits := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			inDigits = true
			n = n*10 + int(c-'0')
			continue
		}
		if inDigits {
			return n
		}
	}
	if inDigits {
		return n
	}
	return 0
}

func ptrInt64(v int64) *int64 { return &v }

func voiceSessionHitsToWire(in []agentcontext.SearchHit) []struct {
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
		}{Path: h.Path, Line: int64(h.Line), Character: int64(h.Character), Preview: h.Preview})
	}
	return out
}

func wireHitsToVoiceSession(in []struct {
	Path      string `json:"path"`
	Line      int64  `json:"line"`
	Character int64  `json:"character"`
	Preview   string `json:"preview"`
},
) []agentcontext.SearchHit {
	out := make([]agentcontext.SearchHit, 0, len(in))
	for _, h := range in {
		out = append(out, agentcontext.SearchHit{
			Path:      h.Path,
			Line:      int(h.Line),
			Character: int(h.Character),
			Preview:   h.Preview,
		})
	}
	return out
}

func (s *TranscriptService) runExecute(params protocol.VoiceTranscriptParams) (protocol.VoiceTranscriptCompletion, bool, string) {
	s.executeMu.Lock()
	defer s.executeMu.Unlock()

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
		vs = voicesession.Load(s.sessions, key, idleReset, &s.ephemeralVoiceSession)
	} else {
		vs = voicesession.Load(s.sessions, key, idleReset, nil)
	}

	if cr := strings.TrimSpace(params.ControlRequest); cr != "" {
		switch cr {
		case "cancel_search":
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.PendingDirectiveApply = nil
			vs.ClarifyQuestion = ""
			vs.ClarifyOriginalTranscript = ""
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Search session closed",
				TranscriptOutcome: "completed",
				UiDisposition:     "hidden",
			}, true, ""
		case "cancel_clarify":
			vs.ClarifyQuestion = ""
			vs.ClarifyOriginalTranscript = ""
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
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

	defaultUiDisposition := func(outcome string, hasActiveSearch bool, success bool) string {
		if !success {
			return "hidden"
		}
		switch strings.TrimSpace(outcome) {
		case "", "completed":
			return "shown"
		case "answer":
			return "hidden"
		case "search", "search_control", "clarify", "clarify_control":
			return "hidden"
		case "irrelevant":
			if hasActiveSearch {
				return "hidden"
			}
			return "skipped"
		default:
			// Be conservative: don't spam UI for unknown outcomes.
			return "hidden"
		}
	}

	// If the daemon is awaiting a clarify answer, interpret cancel/exit and stitch answers here.
	if strings.TrimSpace(vs.ClarifyQuestion) != "" && strings.TrimSpace(vs.ClarifyOriginalTranscript) != "" {
		t := strings.TrimSpace(strings.ToLower(params.Text))
		if t != "" && clarifyControlExitRe.MatchString(t) {
			vs.ClarifyQuestion = ""
			vs.ClarifyOriginalTranscript = ""
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Clarification cancelled",
				TranscriptOutcome: "clarify_control",
				UiDisposition:     "hidden",
			}, true, ""
		}

		// Treat as clarify answer: fold into the original transcript so the executor can continue normally.
		answer := strings.TrimSpace(params.Text)
		if answer != "" {
			params.Text = strings.Join([]string{
				strings.TrimSpace(vs.ClarifyOriginalTranscript),
				"",
				"Clarifying question: " + strings.TrimSpace(vs.ClarifyQuestion),
				"User answer: " + answer,
			}, "\n")
		}
		vs.ClarifyQuestion = ""
		vs.ClarifyOriginalTranscript = ""
	}

	// Async host apply reports are not supported (duplex-only execution). PendingDirectiveApply
	// is consumed immediately after each host.applyDirectives call within this RPC.
	activeFile := strings.TrimSpace(params.ActiveFile)
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

	// If a search hit list is active, interpret lightweight navigation utterances.
	if navKind, ord, ok := parseSearchControl(params.Text); ok && len(vs.SearchResults) > 0 {
		if navKind == "exit" {
			vs.SearchResults = nil
			vs.ActiveSearchIndex = 0
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptCompletion{
				Success:           true,
				Summary:           "Search session closed",
				TranscriptOutcome: "search_control",
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
			TranscriptOutcome: "search_control",
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
		// Apply directives via duplex path just like normal.
		// We bypass executor here, so we manually call host apply once.
		if s.hostApplyClient == nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
		}
		pending := &agentcontext.DirectiveApplyBatch{ID: "search-nav", NumDirectives: len(dirs)}
		vs.PendingDirectiveApply = pending
		hostRes, err := s.hostApplyClient.ApplyDirectives(protocol.HostApplyParams{
			ApplyBatchId: pending.ID,
			ActiveFile:   params.ActiveFile,
			Directives:   dirs,
		})
		_ = hostRes
		_ = err
		vs.PendingDirectiveApply = nil
		if strings.TrimSpace(key) == "" {
			voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
		} else {
			voicesession.SaveKeyed(s.sessions, key, vs)
		}
		return res, true, ""
	}

	maxRepairSteps := config.DefaultMaxRepairSteps
	if dc != nil && dc.MaxTranscriptRepairRpcs != nil {
		maxRepairSteps = int(*dc.MaxTranscriptRepairRpcs)
	}
	if maxRepairSteps < 1 {
		maxRepairSteps = 1
	}

	var lastApplyErr error
	for stepI := 0; stepI < maxRepairSteps; stepI++ {
		vs.Gathered = agentcontext.ApplyGatheredRollingCap(
			vs.Gathered,
			activeFile,
			maxGatheredBytes,
			maxGatheredExcerpts,
		)

		res, dirs, g1, pending, ok, reason := s.executor.Execute(params, vs.Gathered)
		vs.Gathered = g1
		if strings.TrimSpace(key) != "" {
			vs.Gathered = agentcontext.ApplyGatheredRollingCap(vs.Gathered, activeFile, maxGatheredBytes, maxGatheredExcerpts)
		}

		if !ok {
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			if strings.TrimSpace(reason) == "" {
				reason = "executor rejected transcript params"
			}
			return protocol.VoiceTranscriptCompletion{Success: false}, false, reason
		}

		// When a search session is active, irrelevant utterances should not spam "Skipped".
		if res.TranscriptOutcome == "irrelevant" && len(vs.SearchResults) > 0 {
			res.UiDisposition = "hidden"
		} else if strings.TrimSpace(res.UiDisposition) == "" {
			res.UiDisposition = defaultUiDisposition(res.TranscriptOutcome, len(vs.SearchResults) > 0, res.Success)
		}

		if !res.Success || len(dirs) == 0 {
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			if !res.Success && strings.TrimSpace(reason) != "" {
				return res, ok, reason
			}
			// Persist search results when the daemon returned them (even with no directives).
			if res.TranscriptOutcome == "search" && len(res.SearchResults) > 0 {
				vs.SearchResults = wireHitsToVoiceSession(res.SearchResults)
				if res.ActiveSearchIndex != nil {
					vs.ActiveSearchIndex = int(*res.ActiveSearchIndex)
				} else {
					vs.ActiveSearchIndex = 0
				}
				if strings.TrimSpace(key) == "" {
					voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
				} else {
					voicesession.SaveKeyed(s.sessions, key, vs)
				}
			}
			// Persist clarify state so follow-up utterances can be interpreted on the daemon.
			if res.TranscriptOutcome == "clarify" && strings.TrimSpace(res.Summary) != "" {
				vs.ClarifyQuestion = strings.TrimSpace(res.Summary)
				vs.ClarifyOriginalTranscript = strings.TrimSpace(params.Text)
				if strings.TrimSpace(key) == "" {
					voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
				} else {
					voicesession.SaveKeyed(s.sessions, key, vs)
				}
			}
			return res, ok, ""
		}

		if pending == nil || s.hostApplyClient == nil {
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "daemon has directives but no host apply client is configured"
		}

		vs.PendingDirectiveApply = pending
		hostRes, err := s.hostApplyClient.ApplyDirectives(protocol.HostApplyParams{
			ApplyBatchId: pending.ID,
			ActiveFile:   params.ActiveFile,
			Directives:   dirs,
		})
		if err != nil {
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "host.applyDirectives failed: " + err.Error()
		}
		if err := voicesession.ConsumeHostApplyReport(pending.ID, hostRes.Items, &vs); err != nil {
			lastApplyErr = err
			// Retry only for explicit stale-range failures.
			if strings.Contains(err.Error(), "stale_range") && stepI+1 < maxRepairSteps {
				continue
			}
			vs.PendingDirectiveApply = nil
			if strings.TrimSpace(key) == "" {
				voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
			} else {
				voicesession.SaveKeyed(s.sessions, key, vs)
			}
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply failed: " + err.Error()
		}

		// Persist search results when the daemon returned them (even with directives).
		if res.TranscriptOutcome == "search" && len(res.SearchResults) > 0 {
			vs.SearchResults = wireHitsToVoiceSession(res.SearchResults)
			if res.ActiveSearchIndex != nil {
				vs.ActiveSearchIndex = int(*res.ActiveSearchIndex)
			} else {
				vs.ActiveSearchIndex = 0
			}
		}

		if strings.TrimSpace(key) == "" {
			voicesession.StoreEphemeralVoiceSession(&s.ephemeralVoiceSession, vs)
		} else {
			voicesession.SaveKeyed(s.sessions, key, vs)
		}
		return res, ok, ""
	}

	if lastApplyErr != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply failed: " + lastApplyErr.Error()
	}
	return protocol.VoiceTranscriptCompletion{Success: false}, true, "maxTranscriptRepairRpcs exceeded"
}
