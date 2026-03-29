package executor

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents"
	"vocoding.net/vocode/v2/apps/daemon/internal/intents/dispatch/edit"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols"
	"vocoding.net/vocode/v2/apps/daemon/internal/symbols/tags"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

func appendSourceIntentForDirective(dst *[]intents.Intent, intent intents.Intent) {
	*dst = append(*dst, intent)
}

func newDirectiveApplyBatchID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func resolveHostCursorSymbol(r symbols.Resolver, params protocol.VoiceTranscriptParams) *agentcontext.CursorSymbol {
	if r == nil {
		return nil
	}
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
	byteCol, err := tags.ByteOffsetForLineAndUTF16Column(active, line, char)
	if err != nil {
		return nil
	}
	ref, ok := r.ResolveInnermostAtLine(strings.TrimSpace(params.WorkspaceRoot), active, line, byteCol)
	if !ok {
		return nil
	}
	return &agentcontext.CursorSymbol{ID: ref.ID, Name: ref.Name, Kind: ref.Kind}
}

func buildEditExecutionContext(params protocol.VoiceTranscriptParams, ex *intents.ExecutableIntent) (edit.EditExecutionContext, string) {
	if ex == nil {
		return edit.EditExecutionContext{}, "missing executable intent"
	}
	if ex.Kind == intents.ExecutableIntentKindUndo {
		return edit.EditExecutionContext{}, ""
	}
	active := strings.TrimSpace(params.ActiveFile)
	workspaceRoot := strings.TrimSpace(params.WorkspaceRoot)
	if ex.Kind == intents.ExecutableIntentKindEdit && active == "" {
		return edit.EditExecutionContext{}, "activeFile is required when the next intent is an edit"
	}
	if ex.Kind == intents.ExecutableIntentKindEdit && workspaceRoot == "" {
		return edit.EditExecutionContext{}, "workspaceRoot is required when the next intent is an edit"
	}
	fileText := ""
	if active != "" {
		b, err := os.ReadFile(active)
		if err != nil {
			return edit.EditExecutionContext{}, fmt.Sprintf("read active file: %v", err)
		}
		fileText = string(b)
	}
	return edit.EditExecutionContext{
		Instruction:   params.Text,
		ActiveFile:    params.ActiveFile,
		FileText:      fileText,
		WorkspaceRoot: workspaceRoot,
	}, ""
}

func appendGatheredNote(g agentcontext.Gathered, note string) agentcontext.Gathered {
	const maxNotes = 8
	note = strings.TrimSpace(note)
	if note == "" {
		return g
	}
	g.Notes = append(g.Notes, note)
	if len(g.Notes) > maxNotes {
		g.Notes = g.Notes[len(g.Notes)-maxNotes:]
	}
	return g
}
