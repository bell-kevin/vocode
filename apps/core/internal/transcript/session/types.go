package session

// BasePhase is the single active "surface" in the UX model (selection/search vs file browsing),
// with an independent optional clarify overlay suspended on top.
type BasePhase string

const (
	BasePhaseMain          BasePhase = "main"
	BasePhaseSelection     BasePhase = "selection"
	BasePhaseFileSelection BasePhase = "file_selection"
)

// ClarifyOverlay is stored in session when a clarification prompt is active.
// It is suspended over the current BasePhase.
type ClarifyOverlay struct {
	TargetResolution   string
	Question           string
	OriginalTranscript string
}
