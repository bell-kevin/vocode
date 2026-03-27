package protocol

func NewNavigationDirectiveSuccess(action NavigationAction) NavigationDirective {
	return NavigationDirective{
		Kind:   "success",
		Action: &action,
	}
}

func NewNavigationDirectiveNoop(reason string) NavigationDirective {
	return NavigationDirective{
		Kind:   "noop",
		Reason: reason,
	}
}
