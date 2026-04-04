package searchapply

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/search"
	"vocoding.net/vocode/v2/apps/core/internal/transcript/session"
	"vocoding.net/vocode/v2/apps/core/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

const (
	contentSearchMaxHits     = 20
	fileSearchMaxUniquePaths = 20
)

// HostApplyClient applies navigation/edit directives on the host (e.g. VS Code).
type HostApplyClient interface {
	ApplyDirectives(protocol.HostApplyParams) (protocol.HostApplyResult, error)
}

// TranscriptSearch runs workspace content search and file-path search for transcript handling.
type TranscriptSearch struct {
	HostApply             HostApplyClient
	ExtensionHost         workspaceSymbolHost
	NewBatchID            func() string
	NavigateHitDirectives func(params protocol.VoiceTranscriptParams, path string, line0, char0, length int) []protocol.VoiceTranscriptDirective
}

// workspaceSymbolHost is satisfied by [rpc.ExtensionHost]; kept minimal to avoid import cycles in tests.
type workspaceSymbolHost interface {
	WorkspaceSymbolSearch(protocol.HostWorkspaceSymbolSearchParams) (protocol.HostWorkspaceSymbolSearchResult, error)
}

// SearchFromQuery runs LSP workspace symbol search (when ExtensionHost is set), then ripgrep fallback
// with derived literals. symbolKind is an optional classifier hint forwarded to the host (empty = no LSP kind filter).
func (e *TranscriptSearch) SearchFromQuery(params protocol.VoiceTranscriptParams, q, symbolKind string, vs *session.VoiceSession) (protocol.VoiceTranscriptCompletion, bool, string) {
	q = strings.TrimSpace(q)
	if q == "" {
		return protocol.VoiceTranscriptCompletion{}, false, ""
	}
	root := strings.TrimSpace(workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile))
	if root == "" {
		return protocol.VoiceTranscriptCompletion{
			Success: false,
			Summary: "",
		}, true, "search requires workspaceRoot or activeFile"
	}

	var hits []search.Hit
	if e.ExtensionHost != nil {
		symRes, err := e.ExtensionHost.WorkspaceSymbolSearch(protocol.HostWorkspaceSymbolSearchParams{
			Query:      q,
			SymbolKind: strings.TrimSpace(symbolKind),
		})
		if err == nil && len(symRes.Hits) > 0 {
			hits = workspaceSymbolHitsToSearchHits(symRes)
		}
	}
	if len(hits) == 0 {
		var err error
		hits, err = rgContentSearchFallback(root, q, contentSearchMaxHits)
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "search failed: " + err.Error()
		}
	}

	if len(hits) == 0 {
		return completionWorkspaceSearchNoHits(q), true, ""
	}

	hits = prioritizeActiveFileSearchHits(hits, params.ActiveFile)

	res := completionWorkspaceSearchWithHits(hits, q)

	if e.HostApply == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply client not configured"
	}
	if e.NavigateHitDirectives == nil || e.NewBatchID == nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "search engine not fully configured"
	}
	first := hits[0]
	dirs := e.NavigateHitDirectives(params, first.Path, first.Line0, first.Char0, first.Len)
	batchID := e.NewBatchID()
	if vs != nil {
		vs.PendingDirectiveApply = &session.DirectiveApplyBatch{ID: batchID, NumDirectives: len(dirs)}
	}
	hostRes, err := e.HostApply.ApplyDirectives(protocol.HostApplyParams{
		ApplyBatchId: batchID,
		ActiveFile:   params.ActiveFile,
		Directives:   dirs,
	})
	if err != nil {
		if vs != nil {
			vs.PendingDirectiveApply = nil
		}
		return protocol.VoiceTranscriptCompletion{Success: false}, true, "host.applyDirectives failed: " + err.Error()
	}
	if vs != nil && vs.PendingDirectiveApply != nil {
		if err := vs.PendingDirectiveApply.ConsumeHostApplyReport(batchID, hostRes.Items); err != nil {
			vs.PendingDirectiveApply = nil
			return protocol.VoiceTranscriptCompletion{Success: false}, true, "host apply failed: " + err.Error()
		}
		vs.PendingDirectiveApply = nil
	}

	return res, true, ""
}

// searchHitPathSame reports whether a hit path is the editor's active file (clean paths, ASCII case-fold).
func searchHitPathSame(hitPath, activeFile string) bool {
	activeFile = strings.TrimSpace(activeFile)
	if activeFile == "" {
		return false
	}
	hitPath = strings.TrimSpace(hitPath)
	if hitPath == "" {
		return false
	}
	a := filepath.Clean(activeFile)
	b := filepath.Clean(hitPath)
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(a, b)
}

// prioritizeActiveFileSearchHits moves matches in the active editor file to the front (stable), then
// sorts by path / line / column so the current file is ordered predictably.
func prioritizeActiveFileSearchHits(hits []search.Hit, activeFile string) []search.Hit {
	if len(hits) <= 1 || strings.TrimSpace(activeFile) == "" {
		return hits
	}
	out := append([]search.Hit(nil), hits...)
	sort.SliceStable(out, func(i, j int) bool {
		ai := searchHitPathSame(out[i].Path, activeFile)
		aj := searchHitPathSame(out[j].Path, activeFile)
		if ai != aj {
			return ai
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].Line0 != out[j].Line0 {
			return out[i].Line0 < out[j].Line0
		}
		return out[i].Char0 < out[j].Char0
	})
	return out
}

func hitsToProtocolSearchResults(hits []search.Hit) []protocol.VoiceTranscriptSearchHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]protocol.VoiceTranscriptSearchHit, 0, len(hits))
	for _, h := range hits {
		ml := int64(h.Len)
		if ml <= 0 {
			ml = 1
		}
		p := ml
		out = append(out, protocol.VoiceTranscriptSearchHit{
			Path:        h.Path,
			Line:        int64(h.Line0),
			Character:   int64(h.Char0),
			Preview:     h.Preview,
			MatchLength: &p,
		})
	}
	return out
}

func contentHitKey(h search.Hit) string {
	return h.Path + "\x00" + strconv.Itoa(h.Line0) + "\x00" + strconv.Itoa(h.Char0)
}

// rgContentSearchFallback runs ripgrep with the raw query plus derived needles (camelCase, concat)
// and, if still empty, a case-insensitive pass — so phrases like "delta time" still find deltaTime.
func rgContentSearchFallback(root, q string, maxHits int) ([]search.Hit, error) {
	variants := ContentSearchRgVariants(q)
	if len(variants) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{})
	out := make([]search.Hit, 0, maxHits)
	addPart := func(part []search.Hit) {
		for _, h := range part {
			k := contentHitKey(h)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, h)
			if len(out) >= maxHits {
				return
			}
		}
	}
	for _, needle := range variants {
		part, err := search.FixedStringSearch(root, needle, maxHits)
		if err != nil {
			return nil, err
		}
		addPart(part)
		if len(out) >= maxHits {
			return out[:maxHits], nil
		}
	}
	if len(out) == 0 {
		for _, needle := range variants {
			part, err := search.FixedStringSearchFold(root, needle, maxHits)
			if err != nil {
				return nil, err
			}
			addPart(part)
			if len(out) >= maxHits {
				return out[:maxHits], nil
			}
		}
	}
	if len(out) > maxHits {
		return out[:maxHits], nil
	}
	return out, nil
}

func workspaceSymbolHitsToSearchHits(res protocol.HostWorkspaceSymbolSearchResult) []search.Hit {
	out := make([]search.Hit, 0, len(res.Hits))
	for _, h := range res.Hits {
		ln := int(h.Line)
		ch := int(h.Character)
		ml := int(h.MatchLength)
		if ml <= 0 {
			ml = 1
		}
		out = append(out, search.Hit{
			Path:    h.Path,
			Line0:   ln,
			Char0:   ch,
			Len:     ml,
			Preview: h.Preview,
		})
	}
	return out
}

func completionWorkspaceSearchNoHits(q string) protocol.VoiceTranscriptCompletion {
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("no matches for %q", q),
		UiDisposition: "browse",
		Search: &protocol.VoiceTranscriptWorkspaceSearchState{
			NoHits: true,
		},
	}
}

func completionWorkspaceSearchWithHits(hits []search.Hit, q string) protocol.VoiceTranscriptCompletion {
	wireHits := hitsToProtocolSearchResults(hits)
	var z int64
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       fmt.Sprintf("found %d matches for %q", len(hits), q),
		UiDisposition: "browse",
		Search: &protocol.VoiceTranscriptWorkspaceSearchState{
			Results:     wireHits,
			ActiveIndex: &z,
		},
	}
}
