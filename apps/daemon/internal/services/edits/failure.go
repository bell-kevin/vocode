package edits

// EditBuildFailure represents why an edit could not be compiled into executable
// EditActions.
//
// This is daemon-internal: it is not part of the extension-facing protocol.
type EditBuildFailure struct {
	Code    string
	Message string
}

