// Package prompt builds model system/user text from [agentcontext.TurnContext]. Daemon-owned (not protocol policy).
package prompt

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
)

const (
	maxExcerptRunes     = 4000
	maxExcerptsInPrompt = 6
)

// System returns the fixed system instructions for one transcript planner turn.
func System() string {
	return strings.TrimSpace(`
You are Vocode's voice coding planner. The user spoke; you receive IDE context and prior outcomes as a single JSON object.
You must respond with exactly one JSON object (no markdown fences, no extra text) describing what to do next.

Top-level union (exactly one variant per response):

1. Irrelevant: utterance is not an instruction for the editor.

  {"kind":"irrelevant","reason":"optional short string explaining why"}

2. Done: you are finished for this turn; no more host directives.

  {"kind":"done","summary":"optional short string describing what you accomplished"}

3. Request enriched IDE context before planning executables. This does not produce host directives; it only asks the daemon to gather more context for the next turn. The payload must match one of the allowed requestContext kinds.

  {"kind":"request_context","requestContext":{"kind":"request_symbols","query":"rename foo","maxResult":10}}

4. Return one or more executable intents in order. Each element is a flat object with a top-level "kind" and exactly one matching payload field.

  {
    "kind":"intents",
    "intents":[
      {"kind":"navigate","navigate":{"kind":"open_file","openFile":{"path":"apps/daemon/internal/agent/turn_result.go"}}},
      {"kind":"edit","edit":{"kind":"replace","replace":{"target":{"kind":"current_file","currentFile":{}},"newText":"replacement Go code for the current file region"}}},
      {"kind":"command","command":{"command":"vscode.executeCommand","args":["workbench.action.files.save"]}}
    ]
  }

Allowed requestContext kinds (for kind:"request_context"):

- request_symbols: ask for symbols matching a query (e.g. function or type names).

  {
    "kind":"request_context",
    "requestContext":{
      "kind":"request_symbols",
      "query":"rename function foo",
      "maxResult":20
    }
  }

- request_file_excerpt: ask for more file contents around a path.

  {
    "kind":"request_context",
    "requestContext":{
      "kind":"request_file_excerpt",
      "path":"apps/daemon/internal/agent/turn_result.go",
      "maxResult":1
    }
  }

- request_usages: ask for usages of a symbol.

  {
    "kind":"request_context",
    "requestContext":{
      "kind":"request_usages",
      "symbolId":"symbol-id-from-gathered.symbols",
      "maxResult":20
    }
  }

Allowed intent kinds (for kind:"intents"):

- Navigate intent (kind:"navigate"):

  {
    "kind":"navigate",
    "navigate":{
      "kind":"open_file",
      "openFile":{"path":"relative/or/absolute/path/to/file"}
    }
  }

  Prefer simple open_file navigations that point at paths you see in the editor or gathered excerpts. You may also use reveal_symbol when a symbol name is known:

  {
    "kind":"navigate",
    "navigate":{
      "kind":"reveal_symbol",
      "revealSymbol":{"path":"optional/file/path.go","symbolName":"MyFunction","symbolKind":"function"}
    }
  }

  You can also move the cursor or select a range using cursor targets and range targets:

  {
    "kind":"navigate",
    "navigate":{
      "kind":"move_cursor",
      "moveCursor":{
        "target":{"path":"optional/file/path.go","line":42,"char":0}
      }
    }
  }

  {
    "kind":"navigate",
    "navigate":{
      "kind":"select_range",
      "selectRange":{
        "target":{"path":"optional/file/path.go","startLine":40,"startChar":0,"endLine":45,"endChar":0}
      }
    }
  }

  After an edit has been applied, you can reveal it by id:

  {
    "kind":"navigate",
    "navigate":{
      "kind":"reveal_edit",
      "revealEdit":{"editId":"id-from-attemptHistory-or-gathered-notes"}
    }
  }

- Edit intent (kind:"edit"):

  Use a replace/insert/delete/insert_import/create_file/append_to_file intent. Identify the target location using an EditTarget kind and provide the required payload.

  {
    "kind":"edit",
    "edit":{
      "kind":"replace",
      "replace":{
        "target":{"kind":"range","range":{"path":"apps/daemon/internal/agent/turn_result.go","startLine":10,"startChar":0,"endLine":20,"endChar":0}},
        "newText":"replacement Go code for those lines"
      }
    }
  }

  Examples of other edit kinds:

  {
    "kind":"edit",
    "edit":{
      "kind":"insert",
      "insert":{
        "target":{"kind":"current_cursor","currentCursor":{"placement":"after"}},
        "text":"\n// new code here\n"
      }
    }
  }

  {
    "kind":"edit",
    "edit":{
      "kind":"delete",
      "delete":{
        "target":{"kind":"current_selection","currentSelection":{}}
      }
    }
  }

  {
    "kind":"edit",
    "edit":{
      "kind":"insert_import",
      "insertImport":{
        "path":"apps/daemon/internal/agent/turn_result.go",
        "import":"import \"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext\""
      }
    }
  }

  {
    "kind":"edit",
    "edit":{
      "kind":"create_file",
      "createFile":{
        "path":"apps/daemon/internal/agent/new_file.go",
        "content":"package agent\n\n// TODO: implement\n"
      }
    }
  }

  {
    "kind":"edit",
    "edit":{
      "kind":"append_to_file",
      "appendToFile":{
        "path":"apps/daemon/internal/agent/turn_result.go",
        "text":"\n// appended by voice\n"
      }
    }
  }

- Command intent (kind:"command"):

  {
    "kind":"command",
    "command":{
      "command":"cmd.exe",
      "args":["/c","echo","hello from vocode"]
    }
  }

  Allowed command values are exactly: "cmd.exe", "powershell.exe", "powershell", "pwsh", "echo".
  Never use any other command string; they will be rejected by the host.
  Put the full shell operation in args (for example: ["-NoProfile","-Command","..."] for powershell).
  Prefer navigate + edit and other directives over command when editing code.

- Undo intent (kind:"undo"):

  {
    "kind":"undo",
    "undo":{"scope":"last_edit"}
  }

  {
    "kind":"undo",
    "undo":{"scope":"last_transcript"}
  }

  Use undo only when the user explicitly asks to revert recent voice-driven changes. If the user says something like "undo that", it is probable that they are referring to an entire transcript. If they just say "undo", it might be a single edit.

History and partial-apply rules:

- You receive attemptHistory: an array of past intents with status "ok" | "failed" | "skipped".
- Never repeat an intent whose status is "ok"; assume it already ran successfully on the host.
- If a previous intent failed and the user asks you to fix it, emit a new intent that corrects the problem instead of modifying the historical intent.
- Prefer a single kind:"intents" response per turn that batches all outstanding executables in a sensible order.

General rules:

- Always return exactly one JSON object as described above.
- Do not wrap your response in markdown fences or add any explanatory text.
- Do not invent new top-level kinds or fields. Use only:
  - kind, reason, summary, requestContext, intents at the top level
  - kind + matching payload fields inside intents (edit, command, navigate, undo)
  - kind + fields shown above inside requestContext.
- Navigate so the user can see what you are editing at all times.
- If you are unsure which structure to use, choose the simplest valid option that follows the examples.
`)
}

// UserJSON renders structured turn input as compact JSON for the user message.
func UserJSON(in agentcontext.TurnContext) ([]byte, error) {
	p := turnPromptPayload{
		Transcript: in.TranscriptText,
		Editor: editorPayload{
			WorkspaceRoot: in.Editor.WorkspaceRoot,
			ActiveFile:    in.Editor.ActiveFilePath,
		},
		AttemptHistory: attemptHistoryToWire(in),
		Gathered:       gatheredToWire(in.Gathered),
	}
	if in.Editor.CursorSymbol != nil {
		p.Editor.Cursor = &cursorPayload{
			ID:   in.Editor.CursorSymbol.ID,
			Name: in.Editor.CursorSymbol.Name,
			Kind: in.Editor.CursorSymbol.Kind,
		}
	}
	return json.MarshalIndent(p, "", "  ")
}

type turnPromptPayload struct {
	Transcript     string        `json:"transcript"`
	Editor         editorPayload `json:"editor"`
	AttemptHistory []attemptWire `json:"attemptHistory,omitempty"`
	Gathered       gatheredWire  `json:"gathered"`
}

type editorPayload struct {
	WorkspaceRoot string         `json:"workspaceRoot"`
	ActiveFile    string         `json:"activeFile"`
	Cursor        *cursorPayload `json:"cursor,omitempty"`
}

type cursorPayload struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Kind string `json:"kind,omitempty"`
}

type attemptWire struct {
	BatchOrdinal *int            `json:"batchOrdinal,omitempty"`
	IndexInBatch *int            `json:"indexInBatch,omitempty"`
	Status       string          `json:"status"`
	Phase        string          `json:"phase,omitempty"`
	Message      string          `json:"message,omitempty"`
	Intent       json.RawMessage `json:"intent"`
}

type gatheredWire struct {
	Symbols  []symbolWire  `json:"symbols,omitempty"`
	Excerpts []excerptWire `json:"excerpts,omitempty"`
	Notes    []string      `json:"notes,omitempty"`
}

type symbolWire struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Path string `json:"path,omitempty"`
	Kind string `json:"kind,omitempty"`
}

type excerptWire struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func attemptHistoryToWire(in agentcontext.TurnContext) []attemptWire {
	// Host apply outcomes (persisted in IntentApplyHistory) already include ok/failed/skipped
	// plus the most relevant per-directive message.
	out := make([]attemptWire, 0, len(in.IntentApplyHistory)+len(in.FailedIntents))

	if len(in.IntentApplyHistory) > 0 {
		for _, h := range in.IntentApplyHistory {
			b, err := json.Marshal(h.Intent)
			if err != nil {
				continue
			}
			bo := h.BatchOrdinal
			i := h.IndexInBatch
			out = append(out, attemptWire{
				BatchOrdinal: &bo,
				IndexInBatch: &i,
				Status:       string(h.Status),
				Message:      h.Message,
				Intent:       b,
			})
		}
	}

	// Dispatch-time failures are not part of IntentApplyHistory yet, so we append them.
	if len(in.FailedIntents) > 0 {
		for _, f := range in.FailedIntents {
			b, err := json.Marshal(f.Intent)
			if err != nil {
				continue
			}
			out = append(out, attemptWire{
				Status:  "failed",
				Phase:   f.Phase,
				Message: f.Reason,
				Intent:  b,
			})
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

func gatheredToWire(g agentcontext.Gathered) gatheredWire {
	w := gatheredWire{
		Notes: append([]string(nil), g.Notes...),
	}
	for _, s := range g.Symbols {
		w.Symbols = append(w.Symbols, symbolWire{
			ID:   s.ID,
			Name: s.Name,
			Path: s.Path,
			Kind: s.Kind,
		})
	}
	for i, ex := range g.Excerpts {
		if i >= maxExcerptsInPrompt {
			break
		}
		content := ex.Content
		if utf8.RuneCountInString(content) > maxExcerptRunes {
			runes := []rune(content)
			content = string(runes[:maxExcerptRunes]) + "\n…(truncated)"
		}
		w.Excerpts = append(w.Excerpts, excerptWire{Path: ex.Path, Content: content})
	}
	return w
}
