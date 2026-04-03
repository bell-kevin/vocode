package flows

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
	{ID: "select_file", Description: "User wants to find or select files or folders by path or filename fragment, not by searching inside file contents.", Execution: ExecutionSerialized},
	{ID: "control", Description: "User wants to exit or steer the flow (cancel, go back, stop, etc.).", Execution: ExecutionImmediate},
	{ID: "irrelevant", Description: "Nothing here matches what the user is trying to do in this flow.", Execution: ExecutionImmediate},
}

func rootSpec() Spec {
	rootRoutes := []Route{
		{ID: "question", Description: "User asks a question (not a command).", Execution: ExecutionImmediate},
	}
	return Spec{
		Intro:  "You are Vocode's classifier for the ROOT flow.\n\nThe user is NOT in a sub-flow. Given one voice transcript, choose exactly one route id. You only classify — details are resolved later.",
		Routes: append(globalRoutes, rootRoutes...),
	}
}

func workspaceSelectSpec() Spec {
	wsRoutes := []Route{
		{ID: "workspace_select_control", Description: "User wants to move through the workspace hit list or pick a hit by position or number (next, previous, first, third, go to N, etc.).", Execution: ExecutionImmediate},
		{ID: "edit", Description: "User wants to change code at the current focus or selection (e.g. pass an argument, refactor wording), not to start a new workspace search for a name they mention.", Execution: ExecutionSerialized},
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
		{ID: "create", Description: "User wants to add a new file or folder.", Execution: ExecutionSerialized},
		{ID: "delete", Description: "User wants to delete the selected file or folder.", Execution: ExecutionSerialized},
	}
	return Spec{
		Intro: "You are Vocode's classifier for the SELECT FILE result flow.\nThe user already has a list of search hits (files and folders). Choose exactly one route id. You only classify — details are resolved later.\n\n" +
			"If they ask to find code, a function, symbol, or text inside files (e.g. \"main\", \"main function\", \"deltaTime\"), use workspace_select — not select_file. " +
			"Use select_file only for path or filename fragments (e.g. \"src/api\", \"foo.go\").",
		Routes: append(globalRoutes, fsRoutes...),
	}
}
