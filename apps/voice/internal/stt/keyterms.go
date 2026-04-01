package stt

import "slices"

// RealtimeSTTKeyterms are sent as repeated "keyterms" query parameters on the speech-to-text
// WebSocket (Scribe realtime). ElevenLabs documents keyterms for batch STT; realtime support
// should be verified against current API behavior — if the handshake fails, remove or adjust.
//
// Keep each term ≤50 characters, ≤5 words; total list size within product limits (see ElevenLabs docs).

// sttKeytermsProgramming: general software / language vocabulary. "dot" biases spoken file
// extensions (“config dot go”, “dot ts”) toward the punctuation word instead of homophones.
var sttKeytermsProgramming = []string{
	"async",
	"await",
	"boolean",
	"callback",
	"class",
	"closure",
	"commit",
	"compiler",
	"const",
	"constructor",
	"debug",
	"deploy",
	"dot",
	"enum",
	"export",
	"function",
	"generic",
	"GitHub",
	"import",
	"interface",
	"JavaScript",
	"JSON",
	"lambda",
	"namespace",
	"null",
	"package",
	"promise",
	"Python",
	"React",
	"refactor",
	"regex",
	"repository",
	"Rust",
	"slash",
	"snippet",
	"SQL",
	"string",
	"TypeScript",
	"undefined",
	"VS Code",
	"websocket",
}

// sttKeytermsVocodeProduct: product name and branding.
var sttKeytermsVocodeProduct = []string{
	"Vocode",
	"Vocoding",
}

// sttKeytermsVocodeDirectives: bias STT toward generic editor / coding-assistant vocabulary, not
// Vocode-specific UI strings (transcript panel, voice toggle, command palette by name, etc.).
// Short stems + a few common phrases; synonym routing belongs downstream.
var sttKeytermsVocodeDirectives = []string{
	// canonical phrases
	"go to line",
	"go to definition",
	"undo that",
	// stems (reuse across phrasings)
	"find",
	"look for",
	"goto",
	"undo",
	"redo",
	"revert",
	"rollback",
	"line",
	"last",
	"edit",
	"change",
	"open",
	"save",
	"quit",
	"close",
	"cancel",
	"done",
	"format",
	"reveal",
	"explain",
	"selection",
	"replace",
	"tests",
	"references",
	"definition",
	"symbol",
	"terminal",
	"problems",
}

// RealtimeSTTKeyterms is the concatenation of the lists above, in order.
// AllRealtimeSTTKeyterms() appends workspace extras from VOCODE_STT_KEYTERMS_JSON (see workspace_keyterms.go).
var RealtimeSTTKeyterms = slices.Concat(
	sttKeytermsProgramming,
	sttKeytermsVocodeProduct,
	sttKeytermsVocodeDirectives,
)
