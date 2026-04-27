package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/foreverfl/gitt/internal/gitx"
	"github.com/foreverfl/gitt/internal/paths"
	"github.com/foreverfl/gitt/internal/worktree"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old-branch> <new-branch>",
	Short: "Rename a worktree's branch and folder together",
	Long: "Renames the branch <old-branch> to <new-branch>, moves\n" +
		"<repo>/.worktrees/<old-branch> to <repo>/.worktrees/<new-branch>,\n" +
		"and updates the daemon's record. Slash-bearing branch names like\n" +
		"feat/foo are folder-sanitised to feat-foo automatically.",
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDaemon(); err != nil {
			return err
		}
		oldBranch, newBranch := args[0], args[1]
		if oldBranch == newBranch {
			return errors.New("old and new branch are the same")
		}

		mainRoot, err := gitx.MainRepoRoot()
		if err != nil {
			return err
		}

		sockpath, err := paths.SockPath()
		if err != nil {
			return err
		}
		response, err := daemon.Call(sockpath, daemon.Request{
			Op: daemon.OpRenameWorktree,
			Args: map[string]any{
				"repo_root":  mainRoot,
				"old_branch": oldBranch,
				"new_branch": newBranch,
			},
		})
		if err != nil {
			return err
		}
		if !response.OK {
			return fmt.Errorf("rename failed: %s", response.Error)
		}

		newPath := worktree.Path(mainRoot, newBranch)
		fmt.Printf("renamed\n  branch: %s -> %s\n  path:   %s\n", oldBranch, newBranch, newPath)

		cwd, _ := os.Getwd()
		oldPath := worktree.Path(mainRoot, oldBranch)
		if cwd == oldPath {
			fmt.Printf("\nYour shell is still at the old path. cd here:\n  cd %s\n", newPath)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
}
