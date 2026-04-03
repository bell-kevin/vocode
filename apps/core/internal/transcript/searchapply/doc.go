// Package searchapply runs workspace searches for a voice transcript turn: ripgrep-backed queries,
// VoiceTranscriptCompletion payloads, optional session mutation, and HostApply to open the first hit.
//
// SearchFromQuery and FileSearchFromQuery return (completion, handled, reason).
// If handled is false, the query was empty or unused; the caller may fall through.
// If handled is true, this package owned the path; reason is empty on success, else an error message.
//
// Files: transcript_search.go (workspace content hits), file_path_search.go (path list),
// file_search_wire.go (FileSelection protocol builders).
package searchapply
