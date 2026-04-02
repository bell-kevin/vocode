package executor

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// searchLikeQueryFromText extracts a literal ripgrep query when the utterance is clearly workspace search.
func searchLikeQueryFromText(text string) (string, bool) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", false
	}
	lower := strings.ToLower(t)
	prefixes := []string{
		"search for ",
		"find ",
		"search ",
		"where is ",
		"where's ",
		"locate ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			q := strings.TrimSpace(t[len(p):])
			if q == "" {
				return "", false
			}
			return q, true
		}
	}
	return "", false
}

func workspaceSearch(
	params protocol.VoiceTranscriptParams,
	gatheredIn agentcontext.Gathered,
	query string,
) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, agentcontext.Gathered, *agentcontext.DirectiveApplyBatch, bool, string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, "empty search query"
	}
	root := workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile)
	if strings.TrimSpace(root) == "" {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, "search requires workspaceRoot or activeFile"
	}
	hits, err := rgSearch(root, query, 20)
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("search failed: %v", err)
	}
	if len(hits) == 0 {
		return protocol.VoiceTranscriptCompletion{Success: true, Summary: fmt.Sprintf("no matches for %q", query), TranscriptOutcome: "search", UiDisposition: "hidden"}, nil, gatheredIn, nil, true, ""
	}

	first := hits[0]
	wireHits := make([]struct {
		Path      string `json:"path"`
		Line      int64  `json:"line"`
		Character int64  `json:"character"`
		Preview   string `json:"preview"`
	}, 0, len(hits))
	for _, h := range hits {
		wireHits = append(wireHits, struct {
			Path      string `json:"path"`
			Line      int64  `json:"line"`
			Character int64  `json:"character"`
			Preview   string `json:"preview"`
		}{
			Path:      h.Path,
			Line:      int64(h.Line0),
			Character: int64(h.Char0),
			Preview:   h.Preview,
		})
	}

	open := protocol.VoiceTranscriptDirective{
		Kind: "navigate",
		NavigationDirective: &protocol.NavigationDirective{
			Kind: "success",
			Action: &protocol.NavigationAction{
				Kind: "open_file",
				OpenFile: &struct {
					Path string `json:"path"`
				}{Path: first.Path},
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
						Path:      first.Path,
						StartLine: int64(first.Line0),
						StartChar: int64(first.Char0),
						EndLine:   int64(first.Line0),
						EndChar:   int64(first.Char0 + first.Len),
					},
				},
			},
		},
	}
	batchID, err := newDirectiveApplyBatchID()
	if err != nil {
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, gatheredIn, nil, true, fmt.Sprintf("failed to create applyBatchId: %v", err)
	}
	pending := &agentcontext.DirectiveApplyBatch{ID: batchID, NumDirectives: 2}
	z := int64(0)
	return protocol.VoiceTranscriptCompletion{
		Success:           true,
		Summary:           fmt.Sprintf("found %d matches for %q; opened first", len(hits), query),
		TranscriptOutcome: "selection",
		UiDisposition:     "hidden",
		SearchResults:     wireHits,
		ActiveSearchIndex: &z,
	}, []protocol.VoiceTranscriptDirective{open, sel}, gatheredIn, pending, true, ""
}

type rgHit struct {
	Path    string
	Line0   int
	Char0   int
	Len     int
	Preview string
}

func rgBinary() string {
	if p := strings.TrimSpace(os.Getenv("VOCODE_RG_BIN")); p != "" {
		return p
	}
	return "rg"
}

var rgLineRe = regexp.MustCompile(`^(.*):(\d+):(\d+):(.*)$`)

func rgSearch(root, query string, maxHits int) ([]rgHit, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if maxHits <= 0 {
		maxHits = 10
	}
	cmd := exec.Command(rgBinary(), "--column", "-n", "--fixed-strings", query, root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil && stdout.Len() == 0 {
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 1 {
			return nil, nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	out := make([]rgHit, 0, maxHits)
	sc := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		m := rgLineRe.FindStringSubmatch(line)
		if len(m) != 5 {
			continue
		}
		path := filepath.Clean(strings.TrimSpace(m[1]))
		ln1 := atoiSafe(m[2])
		col1 := atoiSafe(m[3])
		if ln1 <= 0 || col1 <= 0 {
			continue
		}
		out = append(out, rgHit{
			Path:    path,
			Line0:   ln1 - 1,
			Char0:   col1 - 1,
			Len:     len(query),
			Preview: strings.TrimSpace(m[4]),
		})
		if len(out) >= maxHits {
			break
		}
	}
	return out, nil
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}
