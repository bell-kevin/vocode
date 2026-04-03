package run

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"vocoding.net/vocode/v2/apps/daemon/internal/agentcontext"
	"vocoding.net/vocode/v2/apps/daemon/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

var (
	fileDeleteRe  = regexp.MustCompile(`(?i)\b(delete|remove|trash)\b`)
	fileOpenRe    = regexp.MustCompile(`(?i)\b(open|show|reveal)\b`)
	fileRenameRe  = regexp.MustCompile(`(?i)\brename\s+(?:to\s+)?(\S+)\s*$`)
	fileMoveRe    = regexp.MustCompile(`(?i)^move\s+to\s+(.+)$`)
	fileCreateFRe = regexp.MustCompile(`(?i)\bcreate\s+file\s+(\S+)\s*$`)
	fileCreateDRe = regexp.MustCompile(`(?i)\bcreate\s+folder\s+(\S+)\s*$`)
)

// HandleFileSelectionUtterance implements file-selection flow: file list navigation and workspace directives.
func HandleFileSelectionUtterance(
	params protocol.VoiceTranscriptParams,
	vs *agentcontext.VoiceSession,
) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, string) {
	root := workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile)
	root = filepath.Clean(root)
	if root == "" {
		return protocol.VoiceTranscriptCompletion{}, nil, "file selection requires workspaceRoot or activeFile"
	}
	if err := ensureFileSelectionPaths(vs, root); err != nil {
		return protocol.VoiceTranscriptCompletion{}, nil, "list workspace files: " + err.Error()
	}
	paths := vs.FileSelectionPaths
	if len(paths) == 0 {
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "Workspace has no files to navigate",
			UiDisposition: "hidden",
		}, nil, ""
	}

	focus := resolveFileSelectionFocus(params, vs, root)
	if focus == "" {
		focus = paths[0]
		vs.FileSelectionIndex = 0
	} else if i := indexOfPath(paths, focus); i >= 0 {
		vs.FileSelectionIndex = i
	} else {
		vs.FileSelectionIndex = 0
		focus = paths[0]
	}
	vs.FileSelectionFocus = focus

	text := strings.TrimSpace(params.Text)
	tl := strings.ToLower(text)

	// List navigation over the flat file list.
	if navKind, ord, ok := parseSelectionNav(text); ok && navKind != "exit" {
		switch navKind {
		case "next":
			if vs.FileSelectionIndex < len(paths)-1 {
				vs.FileSelectionIndex++
			}
		case "back":
			if vs.FileSelectionIndex > 0 {
				vs.FileSelectionIndex--
			}
		case "pick":
			if ord >= 1 && ord <= len(paths) {
				vs.FileSelectionIndex = ord - 1
			}
		}
		focus = paths[vs.FileSelectionIndex]
		vs.FileSelectionFocus = focus
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "file focus updated",
			UiDisposition: "hidden",
			FileSelection: fileSearchStateFromPaths(paths, vs.FileSelectionIndex),
		}, nil, ""
	}

	if fileOpenRe.MatchString(tl) && !fileDeleteRe.MatchString(tl) {
		return openFileDirectiveCompletion(vs, paths, focus)
	}
	if fileDeleteRe.MatchString(tl) {
		if focus == "" {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, "no focused path for delete"
		}
		st, err := os.Stat(focus)
		if err != nil {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, "delete: " + err.Error()
		}
		if st.IsDir() {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, "delete folder not supported in v1; delete a file"
		}
		d := protocol.VoiceTranscriptDirective{
			Kind:                "delete_file",
			DeleteFileDirective: &protocol.DeleteFileDirective{Path: focus},
		}
		vs.FileSelectionPaths = nil
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "delete file",
			UiDisposition: "shown",
		}, []protocol.VoiceTranscriptDirective{d}, ""
	}
	if m := fileRenameRe.FindStringSubmatch(text); len(m) == 2 {
		newName := strings.TrimSpace(m[1])
		if newName == "" {
			return clarifyFile(agentcontext.ClarifyTargetRename, "What should the new name be?")
		}
		parent := filepath.Dir(focus)
		dest := filepath.Join(parent, filepath.Base(newName))
		if destJ, ok := jailUnderWorkspaceRoot(root, dest); ok {
			dest = destJ
		} else {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, "rename escapes workspace root"
		}
		d := protocol.VoiceTranscriptDirective{
			Kind: "move_path",
			MovePathDirective: &protocol.MovePathDirective{
				From: focus,
				To:   dest,
			},
		}
		vs.FileSelectionFocus = dest
		vs.FileSelectionPaths = nil
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "rename (move)",
			UiDisposition: "shown",
			FileSelection: fileSearchStateFromSinglePath(dest),
		}, []protocol.VoiceTranscriptDirective{d}, ""
	}
	if m := fileMoveRe.FindStringSubmatch(text); len(m) == 2 {
		destRaw := strings.TrimSpace(m[1])
		var dest string
		if filepath.IsAbs(destRaw) {
			dest = destRaw
		} else {
			dest = filepath.Join(root, destRaw)
		}
		dest = filepath.Clean(dest)
		if dj, ok := jailUnderWorkspaceRoot(root, dest); ok {
			dest = dj
		} else {
			return clarifyFile(agentcontext.ClarifyTargetMove, "Say a folder path under the workspace to move into.")
		}
		base := filepath.Base(focus)
		destFile := filepath.Join(dest, base)
		if destFile, ok := jailUnderWorkspaceRoot(root, destFile); ok {
			d := protocol.VoiceTranscriptDirective{
				Kind: "move_path",
				MovePathDirective: &protocol.MovePathDirective{
					From: focus,
					To:   destFile,
				},
			}
			vs.FileSelectionFocus = destFile
			vs.FileSelectionPaths = nil
			return protocol.VoiceTranscriptCompletion{
				Success:       true,
				Summary:       "move file",
				UiDisposition: "shown",
				FileSelection: fileSearchStateFromSinglePath(destFile),
			}, []protocol.VoiceTranscriptDirective{d}, ""
		}
		return protocol.VoiceTranscriptCompletion{Success: false}, nil, "move destination escapes workspace"
	}
	if m := fileCreateFRe.FindStringSubmatch(text); len(m) == 2 {
		name := strings.TrimSpace(m[1])
		if name == "" {
			return clarifyFile(agentcontext.ClarifyTargetCreateFile, "What file name should I create?")
		}
		name = filepath.Clean(name)
		var newPath string
		if filepath.IsAbs(name) {
			newPath = name
		} else if strings.Contains(name, string(filepath.Separator)) {
			newPath = filepath.Join(root, name)
		} else {
			newPath = filepath.Join(filepath.Dir(focus), name)
		}
		newPath = filepath.Clean(newPath)
		if np, ok := jailUnderWorkspaceRoot(root, newPath); ok {
			newPath = np
		} else {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, "create_file path escapes workspace"
		}
		ed := protocol.EditDirective{
			Kind: "success",
			Actions: []protocol.EditAction{
				{
					Kind:    "create_file",
					Path:    newPath,
					Content: "",
				},
			},
		}
		d := protocol.VoiceTranscriptDirective{Kind: "edit", EditDirective: &ed}
		vs.FileSelectionFocus = newPath
		vs.FileSelectionPaths = nil
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "create file",
			UiDisposition: "shown",
			FileSelection: fileSearchStateFromSinglePath(newPath),
		}, []protocol.VoiceTranscriptDirective{d}, ""
	}
	if m := fileCreateDRe.FindStringSubmatch(text); len(m) == 2 {
		name := strings.TrimSpace(m[1])
		if name == "" {
			return clarifyFile(agentcontext.ClarifyTargetCreateFolder, "What folder name should I create?")
		}
		name = filepath.Clean(name)
		var newPath string
		if filepath.IsAbs(name) {
			newPath = name
		} else if strings.Contains(name, string(filepath.Separator)) {
			newPath = filepath.Join(root, name)
		} else {
			newPath = filepath.Join(filepath.Dir(focus), name)
		}
		newPath = filepath.Clean(newPath)
		if np, ok := jailUnderWorkspaceRoot(root, newPath); ok {
			newPath = np
		} else {
			return protocol.VoiceTranscriptCompletion{Success: false}, nil, "create_folder path escapes workspace"
		}
		d := protocol.VoiceTranscriptDirective{
			Kind:                  "create_folder",
			CreateFolderDirective: &protocol.CreateFolderDirective{Path: newPath},
		}
		vs.FileSelectionPaths = nil
		return protocol.VoiceTranscriptCompletion{
			Success:       true,
			Summary:       "create folder",
			UiDisposition: "shown",
		}, []protocol.VoiceTranscriptDirective{d}, ""
	}

	if strings.Contains(tl, "move") || strings.Contains(tl, "rename") {
		return clarifyFile(agentcontext.ClarifyTargetMove, "Say the destination path or new name.")
	}
	if strings.Contains(tl, "create") {
		return clarifyFile(agentcontext.ClarifyTargetCreateFile, "Say create file or create folder and the name.")
	}

	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		UiDisposition: "hidden",
	}, nil, ""
}

func clarifyFile(target, q string) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, string) {
	if err := agentcontext.ValidateClarifyTargetResolution(agentcontext.FlowKindFileSelection, target); err != nil {
		return protocol.VoiceTranscriptCompletion{}, nil, err.Error()
	}
	return protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       q,
		UiDisposition: "hidden",
		Clarify:       &protocol.VoiceTranscriptClarifyOffer{TargetResolution: target},
	}, nil, ""
}

func openFileDirectiveCompletion(
	vs *agentcontext.VoiceSession,
	paths []string,
	path string,
) (protocol.VoiceTranscriptCompletion, []protocol.VoiceTranscriptDirective, string) {
	d := protocol.VoiceTranscriptDirective{
		Kind: "navigate",
		NavigationDirective: &protocol.NavigationDirective{
			Kind: "success",
			Action: &protocol.NavigationAction{
				Kind: "open_file",
				OpenFile: &struct {
					Path string `json:"path"`
				}{Path: path},
			},
		},
	}
	c := protocol.VoiceTranscriptCompletion{
		Success:       true,
		Summary:       "open file",
		UiDisposition: "hidden",
	}
	if len(paths) > 0 {
		c.FileSelection = fileSearchStateFromPaths(paths, vs.FileSelectionIndex)
	} else {
		c.FileSelection = fileSearchStateFromSinglePath(path)
	}
	return c, []protocol.VoiceTranscriptDirective{d}, ""
}
