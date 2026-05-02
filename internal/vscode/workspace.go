// Package vscode owns the VSCode multi-root workspace file that gitt
// generates from registered worktrees: discovery via the daemon, and the
// shape of the .code-workspace JSON document on disk.
package vscode

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/foreverfl/gitt/internal/daemon/client"
)

// Folder is one entry under the "folders" key of a .code-workspace file.
// The Name field controls how VSCode labels the folder in its sidebar and
// title bar; setting it per-branch is the whole point of this package, since
// every gitt worktree otherwise shows up as ".worktrees" or its safe-branch
// dir name and is hard to tell apart across multiple windows.
type Folder struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Folders fetches the worktree rows belonging to mainRoot and returns folder
// entries with paths relative to mainRoot so the workspace file is portable
// across machines.
func Folders(mainRoot string) ([]Folder, error) {
	worktrees, err := client.ListWorktreesForRepo(mainRoot)
	if err != nil {
		if errors.Is(err, client.ErrNotRunning) {
			return nil, fmt.Errorf("gitt daemon is not running. start it first: gitt on")
		}
		return nil, err
	}

	folders := make([]Folder, 0, len(worktrees))
	for _, w := range worktrees {
		path, err := filepath.Rel(mainRoot, w.WorktreePath)
		if err != nil {
			path = w.WorktreePath
		}
		folders = append(folders, Folder{Name: w.BranchName, Path: path})
	}
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].Name < folders[j].Name
	})
	return folders, nil
}

// WriteWorkspace replaces only the "folders" key of an existing workspace
// file, preserving any user-edited "settings", "extensions", etc. When the
// file does not yet exist, it writes a skeleton that pre-disables the
// "terminal will be relaunched" prompt that VSCode raises whenever an
// extension contributing terminal env vars sees the workspace structure
// change — gitt rewrites the folder list on every add/remove, so without
// these defaults users get nagged on every command. The defaults are seeded
// once on creation; subsequent calls leave settings alone so the user can
// override or delete them freely.
func WriteWorkspace(workspacePath string, folders []Folder) error {
	doc := map[string]any{}
	fileExisted := true
	if existing, err := os.ReadFile(workspacePath); err == nil {
		if err := json.Unmarshal(existing, &doc); err != nil {
			return fmt.Errorf("parse existing %s: %w", filepath.Base(workspacePath), err)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		fileExisted = false
	} else {
		return fmt.Errorf("read %s: %w", filepath.Base(workspacePath), err)
	}

	doc["folders"] = folders
	if !fileExisted {
		doc["settings"] = map[string]any{
			"terminal.integrated.environmentChangesIndicator": "off",
			"terminal.integrated.environmentChangesRelaunch":  false,
		}
	}

	buf, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encode workspace: %w", err)
	}
	buf = append(buf, '\n')

	if err := os.WriteFile(workspacePath, buf, 0o644); err != nil {
		return fmt.Errorf("write workspace file: %w", err)
	}
	return nil
}
