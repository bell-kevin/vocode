package workspaceselectflow

import "strings"

// looksLikeSingleFileImportLine matches a typical ES/TS module import or export-from line at file scope.
func looksLikeSingleFileImportLine(t string) bool {
	if strings.HasPrefix(t, "import.") || strings.HasPrefix(t, "import(") {
		return false
	}
	if strings.HasPrefix(t, "import") {
		return true
	}
	if strings.HasPrefix(t, "export ") && strings.Contains(t, " from ") {
		return true
	}
	return false
}

// targetTextLooksLikeImportOnlySection is true when every non-empty line in targetText looks like an import/export-from line.
func targetTextLooksLikeImportOnlySection(targetText string) bool {
	lines := strings.Split(targetText, "\n")
	ne := 0
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		ne++
		if !looksLikeSingleFileImportLine(t) {
			return false
		}
	}
	return ne > 0
}

// mergePeelLeadingImportLines strips leading import/export-from lines from replacement when the edit target
// is not itself an import-only region, and returns them so the host can insert them via importLines.
// This recovers when the model embeds imports inside replacementText instead of importLines.
func mergePeelLeadingImportLines(replacement, targetText string) (rest string, peeled []string) {
	if strings.TrimSpace(replacement) == "" {
		return replacement, nil
	}
	if targetTextLooksLikeImportOnlySection(targetText) {
		return replacement, nil
	}
	s := strings.TrimRight(replacement, "\n")
	lines := strings.Split(s, "\n")
	i := 0
	for i < len(lines) {
		raw := lines[i]
		t := strings.TrimSpace(raw)
		if t == "" {
			if len(peeled) == 0 {
				i++
				continue
			}
			break
		}
		if looksLikeSingleFileImportLine(t) {
			peeled = append(peeled, strings.TrimRight(raw, "\r"))
			i++
			continue
		}
		break
	}
	if len(peeled) == 0 {
		return replacement, nil
	}
	rest = strings.Join(lines[i:], "\n")
	if strings.HasSuffix(replacement, "\n") && rest != "" {
		rest += "\n"
	}
	return rest, peeled
}
