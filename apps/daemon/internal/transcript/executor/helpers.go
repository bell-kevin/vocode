package executor

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/hostcaps"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func newDirectiveApplyBatchID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// NewDirectiveApplyBatchID returns a unique id for a host apply batch (same generator as the executor).
func NewDirectiveApplyBatchID() (string, error) {
	return newDirectiveApplyBatchID()
}

func resolveHostCursorSymbol(symProvider hostcaps.SymbolProvider, params protocol.VoiceTranscriptParams) *agentcontext.CursorSymbol {
	cp := params.CursorPosition
	if cp == nil {
		return nil
	}
	active := strings.TrimSpace(params.ActiveFile)
	if active == "" {
		return nil
	}
	line := int(cp.Line)
	char := int(cp.Character)
	if line < 0 || char < 0 {
		return nil
	}

	syms := symProvider.ActiveFileSymbols(params)
	if len(syms) == 0 || len(params.ActiveFileSymbols) == 0 {
		return nil
	}

	type symRange struct {
		startLine int
		startChar int
		endLine   int
		endChar   int
	}
	contains := func(r symRange) bool {
		if line < r.startLine || line > r.endLine {
			return false
		}
		if line == r.startLine && char < r.startChar {
			return false
		}
		if line == r.endLine && char > r.endChar {
			return false
		}
		return true
	}
	size := func(r symRange) int {
		// Rough size in characters; good enough for “innermost” selection.
		return (r.endLine-r.startLine)*100000 + (r.endChar - r.startChar)
	}

	bestIdx := -1
	bestSize := 0
	for i := range params.ActiveFileSymbols {
		s := params.ActiveFileSymbols[i]
		rng := symRange{
			startLine: int(s.Range.StartLine),
			startChar: int(s.Range.StartChar),
			endLine:   int(s.Range.EndLine),
			endChar:   int(s.Range.EndChar),
		}
		if !contains(rng) {
			continue
		}
		sz := size(rng)
		if bestIdx == -1 || sz < bestSize {
			bestIdx = i
			bestSize = sz
		}
	}
	if bestIdx == -1 {
		return nil
	}
	best := params.ActiveFileSymbols[bestIdx]
	ref := symbols.SymbolRef{
		Path: filepath.Clean(active),
		Line: int(best.SelectionRange.StartLine) + 1,
		Kind: strings.TrimSpace(best.Kind),
		Name: strings.TrimSpace(best.Name),
	}
	ref.ID = symbols.BuildSymbolID(ref)
	return &agentcontext.CursorSymbol{ID: ref.ID, Name: ref.Name, Kind: ref.Kind}
}
