package search

import "strings"

// SearchLikeQueryFromText extracts a literal ripgrep query when the utterance is clearly workspace search.
func SearchLikeQueryFromText(text string) (string, bool) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", false
	}
	lower := strings.ToLower(t)
	prefixes := []string{
		"search for ",
		"find ",
		"search ",
		"where is ",
		"where's ",
		"locate ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			q := strings.TrimSpace(t[len(p):])
			if q == "" {
				return "", false
			}
			return q, true
		}
	}
	return "", false
}

// FileSearchLikeQueryFromText extracts a query when the utterance is clearly a file-path search (not in-file text search).
func FileSearchLikeQueryFromText(text string) (string, bool) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", false
	}
	lower := strings.ToLower(t)
	prefixes := []string{
		"find file ",
		"find files ",
		"file named ",
		"open file ",
		"show file ",
		"locate file ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			q := strings.TrimSpace(t[len(p):])
			if q == "" {
				return "", false
			}
			return q, true
		}
	}
	return "", false
}
