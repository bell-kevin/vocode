package flows

// Global route "create" adds new text in the active editor file (placement resolved in a later step).
// Flow select_file also defines "create_entry": a new file or folder on disk from the path list — not editor content.

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
	{ID: "workspace_select", Description: "User wants to find or select a symbol, identifier, or text in the codebase (by name or contents), not by file path alone. Prefer search_query = that name; optional search_symbol_kind = function, class, variable, etc.", Execution: ExecutionSerialized},
	{ID: "select_file", Description: "User wants to find or select files or folders by file or folder name (basename only), not by path and not by searching inside file contents. search_query is a single name segment (e.g. game.js, Res) — no slashes, no absolute path.", Execution: ExecutionSerialized},
	{ID: "create", Description: "User wants to add new content to the file they have open in the editor (e.g. a function, variable, comment, or append at the end). Where it goes is resolved later. Not a new path on disk.", Execution: ExecutionSerialized},
	{ID: "command", Description: "User wants the assistant to run a terminal/shell action (install dependencies, start the dev server, scaffold a project with npx/pnpm, run tests, git commands, etc.). Not a question about how something works, and not editing the open file buffer.", Execution: ExecutionSerialized},
	{ID: "control", Description: "User wants to exit or steer the flow (cancel, go back, stop, etc.).", Execution: ExecutionImmediate},
	{ID: "irrelevant", Description: "Nothing here matches what the user is trying to do in this flow.", Execution: ExecutionImmediate},
}

func rootSpec() Spec {
	rootRoutes := []Route{
		{ID: "question", Description: "User asks a question (not a command).", Execution: ExecutionImmediate},
	}
	return Spec{
		Intro:  "You are Vocode's classifier for the ROOT flow.\n\nThe user is NOT in a sub-flow. They may have an active editor file and a text selection.\n\nUser JSON may include activeFile for context only — for select_file, never put a path in search_query; output the file or folder basename only (e.g. game.js).\n\nGiven one voice transcript, choose exactly one route id. You only classify — details are resolved later.",
		Routes: append(globalRoutes, rootRoutes...),
	}
}

func workspaceSelectSpec() Spec {
	wsRoutes := []Route{
		{ID: "workspace_select_control", Description: "User wants to move through the workspace hit list or pick a hit by position or number (next, previous, first, third, go to N, etc.).", Execution: ExecutionImmediate},
		{ID: "edit", Description: "User wants to change code at the current focus or selection (e.g. pass an argument, refactor wording, implement an algorithm), not a new workspace search.", Execution: ExecutionSerialized},
		{ID: "rename", Description: "User wants to rename the thing at the current hit or selection to a new name (e.g. \"rename X to Y\", \"call it Z\").", Execution: ExecutionSerialized},
		{ID: "delete", Description: "User wants to delete the current selection or hit.", Execution: ExecutionSerialized},
	}
	return Spec{
		Intro:  "You are Vocode's classifier for the WORKSPACE SELECT flow.\nThe user already has a list of workspace text/symbol search hits. They also have the editor focused with a text selection.\n\nChoose exactly one route id. You only classify — details are resolved later.",
		Routes: append(globalRoutes, wsRoutes...),
	}
}

func fileSelectSpec() Spec {
	fsRoutes := []Route{
		{ID: "file_select_control", Description: "User wants to move through the file hit list or pick a hit by position or number (next, previous, first, third, go to N, etc.).", Execution: ExecutionImmediate},
		{ID: "move", Description: "User wants to move the selected file or folder to a different path.", Execution: ExecutionSerialized},
		{ID: "rename", Description: "User wants to rename the selected file or folder (path/name).", Execution: ExecutionSerialized},
		{ID: "create_entry", Description: "User wants a new file or folder on disk under the selected row (e.g. create/add/make/new or similar). Not editor buffer content.", Execution: ExecutionSerialized},
		{ID: "delete", Description: "User wants to delete the selected file. (Workspace root and folders are not deletable via this route.)", Execution: ExecutionSerialized},
	}
	return Spec{
		Intro: "You are Vocode's classifier for the SELECT FILE result flow.\nThe user already has a list of search hits (files and folders). Choose exactly one route id. You only classify — details are resolved later.\n\n" +
			"workspace_select: find code, symbols, or text inside files. select_file: name a file or folder by basename to look up paths — not contents search.\n\n" +
			"create_entry: new path on disk under the selection (empty search_query). create: open editor buffer only — not for naming a new file in this flow.",
		Routes: append(globalRoutes, fsRoutes...),
	}
}
