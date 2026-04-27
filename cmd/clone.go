package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/foreverfl/gitt/internal/gitx"
	"github.com/foreverfl/gitt/internal/worktreeclient"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <url> [dir]",
	Short: "Clone a repo into gitt's bare layout (<dir>/.bare + .worktrees/<default-branch>)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoURL := args[0]

		var target string
		if len(args) >= 2 {
			target = args[1]
		} else {
			target = gitx.DeriveCloneDir(repoURL)
			if target == "" {
				return fmt.Errorf("could not derive a directory name from %q; pass [dir] explicitly", repoURL)
			}
		}

		absTarget, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("resolve target: %w", err)
		}

		if entries, err := os.ReadDir(absTarget); err == nil && len(entries) > 0 {
			return fmt.Errorf("destination %q already exists and is not empty", absTarget)
		}
		if err := os.MkdirAll(absTarget, 0o755); err != nil {
			return fmt.Errorf("create target dir: %w", err)
		}

		bareDir := filepath.Join(absTarget, ".bare")
		if err := gitx.CloneBare(repoURL, bareDir); err != nil {
			return err
		}

		if err := os.WriteFile(filepath.Join(absTarget, ".git"), []byte("gitdir: ./.bare\n"), 0o644); err != nil {
			return fmt.Errorf("write .git pointer: %w", err)
		}

		defaultBranch, err := gitx.HeadBranchOf(bareDir)
		if err != nil {
			return fmt.Errorf("detect default branch: %w", err)
		}

		worktreePath := filepath.Join(absTarget, ".worktrees", defaultBranch)
		if err := os.MkdirAll(filepath.Dir(worktreePath), 0o755); err != nil {
			return fmt.Errorf("create .worktrees dir: %w", err)
		}
		if err := gitx.WorktreeAdd(absTarget, worktreePath, defaultBranch, false); err != nil {
			return err
		}

		if err := worktreeclient.TryRegister(absTarget, defaultBranch, worktreePath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: daemon registration failed: %v\n", err)
		}

		fmt.Printf("\ncloned\n  project: %s\n  branch:  %s\n  worktree: %s\n", absTarget, defaultBranch, worktreePath)
		fmt.Printf("\nNext:\n  cd %s\n", worktreePath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}