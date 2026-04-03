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
		return selectFileSpec()
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
	{ID: "workspace_select", Description: "Find occurrences in file contents (ripgrep fixed-string / literal needle in source).", Execution: ExecutionSerialized},
	{ID: "select_file", Description: "Find files or folders by path or filename fragment (not by searching inside file bodies).", Execution: ExecutionSerialized},
	{ID: "control", Description: "Flow navigation (such as exit)", Execution: ExecutionImmediate},
	{ID: "irrelevant", Description: "Not actionable in this flow.", Execution: ExecutionImmediate},
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
		{ID: "workspace_select_control", Description: "Navigate the workspace hit list (next/previous, jump/goto by number).", Execution: ExecutionImmediate},
		{ID: "edit", Description: "They want to edit or change code (scoped edit), not just navigate the list.", Execution: ExecutionSerialized},
		{ID: "delete", Description: "They want to delete this selection.", Execution: ExecutionSerialized},
	}
	return Spec{
		Intro:  "You are Vocode's classifier for the WORKSPACE SELECT flow.\nThe user already has a list of workspace text/symbol search hits. Choose exactly one route id. You only classify — details are resolved later.",
		Routes: append(globalRoutes, wsRoutes...),
	}
}

func selectFileSpec() Spec {
	selectFileRoutes := []Route{
		{ID: "file_select_control", Description: "Navigate the file hit list (next/previous, jump/goto by number).", Execution: ExecutionImmediate},
		{ID: "open", Description: "Open the selected file.", Execution: ExecutionSerialized},
		{ID: "move", Description: "Move selected file or folder to a new location.", Execution: ExecutionSerialized},
		{ID: "rename", Description: "Rename selected file or folder.", Execution: ExecutionSerialized},
		{ID: "create", Description: "Create a new file or folder in the selected folder.", Execution: ExecutionSerialized},
		{ID: "delete", Description: "Delete the selected file or folder.", Execution: ExecutionSerialized},
	}
	return Spec{
		Intro:  "You are Vocode's classifier for the SELECT FILE result flow.\nThe user already has a list of search hits (files and folders). Choose exactly one route id. You only classify — details are resolved later.",
		Routes: append(globalRoutes, selectFileRoutes...),
	}
}
