package clarify

import "vocoding.net/vocode/v2/apps/core/internal/transcript/session"

func BaseFlowKindFromPhase(p session.BasePhase) BaseFlowKind {
	switch p {
	case session.BasePhaseSelection:
		return BaseFlowWorkspaceSelect
	case session.BasePhaseFileSelection:
		return BaseFlowSelectFile
	default:
		return BaseFlowMain
	}
}

func ValidateForBasePhase(base session.BasePhase, targetResolution string) error {
	parent := BaseFlowKindFromPhase(base)
	return ValidateClarifyTargetResolution(parent, targetResolution)
}

func BuildClarificationContext(ov session.ClarifyOverlay, answerText string) ClarificationContext {
	return ClarificationContext{
		OriginalTranscript:      ov.OriginalTranscript,
		ClarifyQuestion:         ov.Question,
		AnswerText:              answerText,
		ClarifyTargetResolution: ov.TargetResolution,
	}
}
