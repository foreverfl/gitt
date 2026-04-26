package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/foreverfl/gitt/internal/gitx"
	"github.com/foreverfl/gitt/internal/worktree"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <branch>",
	Short: "Create a new git worktree for <branch>",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDaemon(); err != nil {
			return err
		}
		branch := args[0]

		repoRoot, err := gitx.RepoRoot()
		if err != nil {
			return err
		}
		target := worktree.Path(repoRoot, branch)

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create worktree parent: %w", err)
		}

		exists, err := gitx.BranchExists(branch)
		if err != nil {
			return err
		}

		if err := gitx.WorktreeAdd(target, branch, !exists); err != nil {
			return err
		}

		if exists {
			fmt.Printf("created worktree\n  path:   %s\n  branch: %s\n", target, branch)
		} else {
			fmt.Printf("created worktree (new branch)\n  path:   %s\n  branch: %s\n", target, branch)
		}
		fmt.Printf("\nOpen a new terminal, then run:\n  cd %s\n  # start your AI CLI here\n", target)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
