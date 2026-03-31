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
- search: locate code/text in the workspace (return a searchQuery string)
- question: user is asking a question; return answerText for the UI
- irrelevant: not actionable

Return exactly ONE JSON object with this schema:

{
  "kind": "instruction" | "search" | "question" | "irrelevant",
  "searchQuery"?: string, // required only when kind="search"
  "answerText"?: string   // required only when kind="question"
}

Rules:
- Prefer "instruction" unless the user is clearly searching ("find", "search for", "where is", "locate") or asking a question.
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

