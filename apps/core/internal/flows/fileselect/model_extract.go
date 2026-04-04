package fileselectflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"vocoding.net/vocode/v2/apps/core/internal/agent"
	"vocoding.net/vocode/v2/apps/core/internal/search"
	"vocoding.net/vocode/v2/apps/core/internal/workspace"
	protocol "vocoding.net/vocode/v2/packages/protocol/go"
)

const moveWorkspaceDirHintCap = 400

// workspaceDirectoryHints lists existing directories under root (one and two levels deep) for move-target disambiguation.
func workspaceDirectoryHints(root string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil
	}
	root = filepath.Clean(root)
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	add := func(rel string) {
		rel = strings.TrimSpace(rel)
		rel = strings.Trim(rel, `/\`)
		rel = filepath.ToSlash(rel)
		if rel == "" || strings.Contains(rel, "..") {
			return
		}
		if _, ok := seen[rel]; ok {
			return
		}
		seen[rel] = struct{}{}
		out = append(out, rel)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		name := e.Name()
		if name == "" || strings.HasPrefix(name, ".") {
			continue
		}
		if !e.IsDir() {
			continue
		}
		add(name)
		sub := filepath.Join(root, name)
		subEntries, err := os.ReadDir(sub)
		if err != nil {
			continue
		}
		for _, se := range subEntries {
			n2 := se.Name()
			if n2 == "" || strings.HasPrefix(n2, ".") {
				continue
			}
			if !se.IsDir() {
				continue
			}
			add(filepath.Join(name, n2))
		}
	}
	sort.Strings(out)
	if len(out) > moveWorkspaceDirHintCap {
		out = out[:moveWorkspaceDirHintCap]
	}
	return out
}

// extractRenameBasename uses the model when the transcript does not match "rename … to …".
func extractRenameBasename(ctx context.Context, m agent.ModelClient, params protocol.VoiceTranscriptParams, fromPath, transcript string) (newBaseName, lspNewName string, err error) {
	root := strings.TrimSpace(workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile))
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"newBaseName"},
		"properties": map[string]any{
			"newBaseName": map[string]any{
				"type":        "string",
				"description": "New file or folder base name only (e.g. foo.go, MyComponent.tsx, or a folder name). No path separators.",
			},
			"lspNewName": map[string]any{
				"type":        "string",
				"description": "If this is a code file rename, the new identifier the language server should rename (must be a single identifier like Foo or foo_bar). Empty if unsure.",
			},
		},
	}
	user := map[string]any{
		"transcript":    transcript,
		"selectedPath":  fromPath,
		"workspaceRoot": root,
	}
	userBytes, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return "", "", err
	}
	sys := strings.TrimSpace(`
You extract a rename target from a voice transcript for the selected file or folder path.
Return JSON only: newBaseName is the new basename (keep the extension appropriate for a code file when relevant).
If the user is renaming a source file and you can infer the main exported or primary symbol name after rename, set lspNewName to a valid single identifier for that language; otherwise use "".
`)
	out, err := m.Call(ctx, agent.CompletionRequest{
		System:     sys,
		User:       string(userBytes),
		JSONSchema: schema,
	})
	if err != nil {
		return "", "", err
	}
	var parsed struct {
		NewBaseName string `json:"newBaseName"`
		LspNewName  string `json:"lspNewName"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		return "", "", fmt.Errorf("decode rename extract json: %w", err)
	}
	return search.TrimSttTrailingSentenceDot(strings.TrimSpace(parsed.NewBaseName)), strings.TrimSpace(parsed.LspNewName), nil
}

// extractCreateEntryFileName returns a single new file or folder basename from the transcript (no path segments).
func extractCreateEntryFileName(ctx context.Context, m agent.ModelClient, params protocol.VoiceTranscriptParams, createUnderDir, transcript string) (string, error) {
	root := strings.TrimSpace(workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile))
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"fileName"},
		"properties": map[string]any{
			"fileName": map[string]any{
				"type":        "string",
				"description": "New file or folder name only (e.g. what.js, README.md). Spoken 'dot' must become a period. No slashes, no prose.",
			},
		},
	}
	user := map[string]any{
		"transcript":        transcript,
		"createUnderFolder": filepath.Base(filepath.Clean(strings.TrimSpace(createUnderDir))),
		"workspaceRoot":     root,
	}
	userBytes, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return "", err
	}
	sys := strings.TrimSpace(`
You extract the new file or folder name the user wants to create on disk.
Return JSON only: fileName is one path segment (basename). Map spoken punctuation to real characters (e.g. "dot" or "point" before an extension → period). Strip phrases like "make a file", "create", "called", "named". No directories in fileName — only the new item's name.
`)
	out, err := m.Call(ctx, agent.CompletionRequest{
		System:     sys,
		User:       string(userBytes),
		JSONSchema: schema,
	})
	if err != nil {
		return "", err
	}
	var parsed struct {
		FileName string `json:"fileName"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		return "", fmt.Errorf("decode create entry extract json: %w", err)
	}
	return search.TrimSttTrailingSentenceDot(strings.TrimSpace(parsed.FileName)), nil
}

// extractMoveDestination asks the model for a workspace-relative destination path (not raw transcript slicing).
func extractMoveDestination(ctx context.Context, m agent.ModelClient, params protocol.VoiceTranscriptParams, fromPath, transcript string) (destFragment string, err error) {
	root := strings.TrimSpace(workspace.EffectiveWorkspaceRoot(params.WorkspaceRoot, params.ActiveFile))
	hints := workspaceDirectoryHints(root)
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"destination"},
		"properties": map[string]any{
			"destination": map[string]any{
				"type": "string",
				"description": "Path relative to workspace root (e.g. src/lib). For the workspace root itself, use workspaceRootBasename from the user JSON or \".\". " +
					"No filler words. Absolute only if user clearly asked for a full system path.",
			},
		},
	}
	user := map[string]any{
		"transcript":              transcript,
		"sourcePath":              fromPath,
		"workspaceRoot":           root,
		"workspaceRootBasename":   filepath.Base(filepath.Clean(root)),
		"workspaceSubdirectories": hints,
	}
	userBytes, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return "", err
	}
	sys := strings.TrimSpace(`
You decide where to move the selected file or folder. Return JSON only: { "destination": "<relative path or absolute if they asked>" }.

- Paths are relative to workspaceRoot (slash-separated). Absolute only if they clearly asked for a full system path.
- To mean the workspace root, output workspaceRootBasename exactly as that folder is named (see JSON), or ".". The host treats that name as the root folder.
- For a subfolder, use its path under the root (workspaceSubdirectories lists real names; spelling can be approximate).
- Directory destination only (no trailing source filename). Strip filler words; do not paste sourcePath as destination.
`)
	out, err := m.Call(ctx, agent.CompletionRequest{
		System:     sys,
		User:       string(userBytes),
		JSONSchema: schema,
	})
	if err != nil {
		return "", err
	}
	var parsed struct {
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed); err != nil {
		return "", fmt.Errorf("decode move extract json: %w", err)
	}
	return search.TrimSttTrailingSentenceDot(strings.TrimSpace(parsed.Destination)), nil
}
