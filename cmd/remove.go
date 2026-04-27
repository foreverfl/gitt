package cmd

import (
	"fmt"
	"os"

	"github.com/foreverfl/gitt/internal/gitx"
	"github.com/foreverfl/gitt/internal/worktree"
	"github.com/foreverfl/gitt/internal/worktreeclient"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <branch>",
	Short: "Remove the git worktree for <branch>",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDaemon(); err != nil {
			return err
		}
		branch := args[0]

		mainRoot, err := gitx.MainRepoRoot()
		if err != nil {
			return err
		}
		target := worktree.Path(mainRoot, branch)

		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("no worktree at %s", target)
			}
			return fmt.Errorf("stat worktree: %w", err)
		}

		if err := gitx.WorktreeRemove(target); err != nil {
			fmt.Fprintln(os.Stderr, "tip: if the worktree has uncommitted or untracked changes, commit or stash them first.")
			return err
		}

		if err := worktreeclient.Release(mainRoot, branch); err != nil {
			fmt.Fprintf(os.Stderr, "warning: worktree removed but daemon record cleanup failed: %v\n", err)
		}

		fmt.Printf("removed worktree\n  path:   %s\n  branch: %s\n", target, branch)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
