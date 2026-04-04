package flows

// Global route "create" adds new text in the active editor file (placement resolved in a later step).
// Flow file_select also defines "create_entry": a new file or folder on disk from the path list — not editor content.

// Route is one transcript resolution option within a flow.
type Route struct {
	ID          string
	Description string
	// Execution is host-only ordering metadata; never exposed to the routing model.
	Execution Execution
}

// Spec is the classifier contract for a flow (prompt + allowed route ids).
type Spec struct {
	Intro  string
	Routes []Route
}

// SpecFor returns the classifier spec for the given flow.
func SpecFor(f ID) Spec {
	switch f {
	case WorkspaceSelect:
		return workspaceSelectSpec()
	case SelectFile:
		return fileSelectSpec()
	default:
		return rootSpec()
	}
}

// RouteIDs returns route ids in spec order (matches prompt / JSON enum).
func (s Spec) RouteIDs() []string {
	out := make([]string, len(s.Routes))
	for i, r := range s.Routes {
		out[i] = r.ID
	}
	return out
}

var globalRoutes = []Route{
	{ID: "workspace_select", Description: `Search inside file contents: symbols, identifiers, or literal text (by name or substring). Not for "open …" (that is file_select). Ambiguous with no path-on-disk cue → prefer this over file_select. Go to main" without file/open cues → here. Output: search_query = identifier or literal substring to find, not a prose paraphrase (see global Rules for literal-text exception and search_symbol_kind).`, Execution: ExecutionSerialized},
	{ID: "file_select", Description: "Find or open a file or folder by basename (path on disk). “Open …”, extensions, and obvious file/folder names → here. search_query = single basename segment only — no slashes, no full paths, never paste activeFile (see global Rules for STT “dot” → period). workspaceFolderOpen false is OK; host handles it.", Execution: ExecutionSerialized},
	{ID: "create", Description: "Add new content by inserting at a line boundary in the active editor (no text selected). Only when hasNonemptySelection is false; otherwise classify as edit (or irrelevant). The user must name or clearly imply what to add — e.g. a function, method, variable, class, interface, type, comment, import, test, or block of code. Vague \"add something\" / \"put code here\" with no identifiable what → not create.", Execution: ExecutionSerialized},
	{ID: "command", Description: "Run terminal/shell work now: install, scaffold, git init, run tests/build, dev server, etc. Use for clear execute-now intent, including polite questions like “can you run the tests?” when they mean execution, not an explanation.", Execution: ExecutionSerialized},
	{ID: "control", Description: "Dismiss or leave the current flow only: exit, cancel, go back, stop, quit, never mind, and short synonyms.", Execution: ExecutionImmediate},
	{ID: "irrelevant", Description: "Not actionable in this flow, off-topic, talking to someone else, noise, or nonsensical. Also: thanks/okay/got it (not clearly exit); vague create with no “what”; ROOT + selection + only “fix this”/“make it work” with no named what to add.", Execution: ExecutionImmediate},
}

func rootSpec() Spec {
	rootRoutes := []Route{
		{ID: "question", Description: "Informational intent only: how/what/why, explanations. If they want a command run now (including polite “can you run …?”), that is command, not question.", Execution: ExecutionImmediate},
	}
	return Spec{
		Intro: `You are Vocode's classifier for the ROOT flow. Input is speech-to-text; expect informal phrasing.

The user is not in a sub-flow. User JSON may include activeFile, hasNonemptySelection, workspaceRoot, hostPlatform, workspaceFolderOpen.

Tie-breaks (ROOT):
- Compound utterance (search + create/command in one line): prefer workspace_select or file_select over create or command (search wins this turn).
- control vs thanks/okay: real exit/dismiss → control; casual thanks without leaving → irrelevant (see irrelevant route).

Choose exactly one route. You only classify; details are resolved later.`,
		Routes: append(globalRoutes, rootRoutes...),
	}
}

func workspaceSelectSpec() Spec {
	wsRoutes := []Route{
		{ID: "workspace_select_control", Description: "Only for the current workspace hit list: next, previous, pick by position (e.g. first hit, third result). Not for utterances that name a new symbol or file to find — those are workspace_select or file_select with a fresh search_query.", Execution: ExecutionImmediate},
		{ID: "edit", Description: "Replace or update code in the current selection (or symbol scope when the caret is inside a symbol). Use whenever hasNonemptySelection is true and they want to change the highlight: add state, hooks, handlers, JSX, fix logic, or vague “fix this” / “make it work”. Create is unavailable while text is selected.", Execution: ExecutionSerialized},
		{ID: "rename", Description: "Rename the thing at the current hit or selection (e.g. rename X to Y, call it Z).", Execution: ExecutionSerialized},
		{ID: "delete", Description: "Delete the current selection or hit.", Execution: ExecutionSerialized},
	}
	return Spec{
		Intro: `You are Vocode's classifier for the WORKSPACE SELECT flow: the user has workspace search hits; the editor may have a non-empty selection. Input is speech-to-text. User JSON may include hasNonemptySelection and activeFile. Follow each route's description and the global Rules (especially search_query and workspace_select vs file_select).

Choose exactly one route. You only classify; details are resolved later.`,
		Routes: append(globalRoutes, wsRoutes...),
	}
}

func fileSelectSpec() Spec {
	fsRoutes := []Route{
		{ID: "file_select_control", Description: "Navigate the file hit list (next, previous, pick by number, etc.).", Execution: ExecutionImmediate},
		{ID: "move", Description: "Move the selected file or folder to another path.", Execution: ExecutionSerialized},
		{ID: "rename", Description: "Rename the selected file or folder.", Execution: ExecutionSerialized},
		{ID: "create_entry", Description: "New file or folder on disk under the focused list row (add/make/create/new + a name). Not the editor-buffer create route. search_query must be empty.", Execution: ExecutionSerialized},
		{ID: "delete", Description: "Delete the selected file. (Workspace root and folders are not deletable via this route.)", Execution: ExecutionSerialized},
	}
	return Spec{
		Intro: `You are Vocode's classifier for the SELECT FILE flow: the user has file/folder path hits. Input is speech-to-text. Use route descriptions plus global Rules (search_query, workspace_select vs file_select, create vs create_entry).

Choose exactly one route. You only classify; details are resolved later.`,
		Routes: append(globalRoutes, fsRoutes...),
	}
}
