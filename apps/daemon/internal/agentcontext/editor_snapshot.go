package agentcontext

import (
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

// CursorSymbol is the innermost definition at the caret (tree-sitter), when resolvable.
// In [EditorSnapshot], CursorSymbol is always represented as a pointer: nil means
// "no symbol at this caret" (unambiguous file, tags missing, or position outside any def span)—not "unknown".
type CursorSymbol struct {
	ID   string
	Name string
	Kind string
}

// EditorSnapshot is the IDE view from this voice.transcript RPC params (active path, caret symbol).
// The host sends fresh activeFile/cursor on every RPC; buffer text for that file lives under
// [Gathered.Excerpts] (seeded from disk plus request_context).
// Within a single Execute loop the params snapshot is fixed until the host sends another transcript.
type EditorSnapshot struct {
	WorkspaceRoot  string
	ActiveFilePath string
	CursorSymbol   *CursorSymbol
}

// EditorSnapshotFromParams builds [EditorSnapshot] from RPC params and resolved cursor symbol.
func EditorSnapshotFromParams(p protocol.VoiceTranscriptParams, cursor *CursorSymbol) EditorSnapshot {
	active := strings.TrimSpace(p.ActiveFile)
	s := EditorSnapshot{
		WorkspaceRoot:  workspace.EffectiveWorkspaceRoot(p.WorkspaceRoot, active),
		ActiveFilePath: active,
	}
	if cursor != nil {
		id := strings.TrimSpace(cursor.ID)
		name := strings.TrimSpace(cursor.Name)
		kind := strings.TrimSpace(cursor.Kind)
		if id != "" || name != "" || kind != "" {
			s.CursorSymbol = &CursorSymbol{ID: id, Name: name, Kind: kind}
		}
	}
	return s
}
