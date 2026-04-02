package prompt

import (
	"encoding/json"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

func TranscriptClassifierSystem() string {
	return strings.TrimSpace(`
You are Vocode's first-pass transcript router.

Given a single voice transcript, decide which route it should take:
- instruction: change code or perform an editor operation
- search: locate code/text in the workspace (return a searchQuery string) — opens the selection / match list flow
- file_selection: user wants file or folder operations in the workspace (rename, move, delete, create, open another file by path, "show me files") — not editing code in the current buffer
- question: user is asking a question; return answerText for the UI
- irrelevant: not actionable

Return exactly ONE JSON object with this schema:

{
  "kind": "instruction" | "search" | "file_selection" | "question" | "irrelevant",
  "searchQuery"?: string, // required only when kind="search"
  "answerText"?: string   // required only when kind="question"
}

Rules:
- Prefer "instruction" unless the user is clearly searching, doing file/folder ops, or asking a question.
- Use "file_selection" for delete/move/rename/create file or folder, or "open X.ts" when they mean a path in the project (not find-in-workspace).
- For kind="search": searchQuery should be the literal text to search for (identifiers, strings). Do NOT include filler words.
  - Example: "find the test function" -> searchQuery: "function test" (or "test(" if more appropriate)
- For kind="question": answerText must be concise plain text. Do NOT include markdown fences or extra keys.
- Do not include any extra keys. No markdown.
`)
}

func TranscriptClassifierUserJSON(in agentcontext.TranscriptClassifierContext) ([]byte, error) {
	type payload struct {
		Instruction string `json:"instruction"`
		ActiveFile  string `json:"activeFile"`
		WorkspaceRoot string `json:"workspaceRoot"`
		CursorSymbol *struct {
			Name string `json:"name,omitempty"`
			Kind string `json:"kind,omitempty"`
		} `json:"cursorSymbol,omitempty"`
	}
	var cursor *struct {
		Name string `json:"name,omitempty"`
		Kind string `json:"kind,omitempty"`
	}
	if in.Editor.CursorSymbol != nil {
		cursor = &struct {
			Name string `json:"name,omitempty"`
			Kind string `json:"kind,omitempty"`
		}{Name: in.Editor.CursorSymbol.Name, Kind: in.Editor.CursorSymbol.Kind}
	}
	return json.MarshalIndent(payload{
		Instruction: strings.TrimSpace(in.Instruction),
		ActiveFile:  in.Editor.ActiveFilePath,
		WorkspaceRoot: in.Editor.WorkspaceRoot,
		CursorSymbol: cursor,
	}, "", "  ")
}

