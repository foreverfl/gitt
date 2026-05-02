package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/foreverfl/gitt/internal/daemon/client"
	"github.com/foreverfl/gitt/internal/gitx"
	"github.com/foreverfl/gitt/internal/store/repo"
	"github.com/spf13/cobra"
)

var listGlobal bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List gitt-managed worktrees for the current repo",
	Long: "By default lists worktrees belonging to the repository that\n" +
		"contains the current working directory. Pass --global / -g to\n" +
		"list every worktree the daemon knows about, across all repos.",
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDaemon(); err != nil {
			return err
		}

		var (
			worktrees []repo.Worktree
			err       error
		)
		if listGlobal {
			worktrees, err = client.ListWorktrees()
		} else {
			mainRoot, mainErr := gitx.MainRepoRoot()
			if mainErr != nil {
				return fmt.Errorf("%w (or pass --global to list every repo)", mainErr)
			}
			worktrees, err = client.ListWorktreesForRepo(mainRoot)
		}
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		if len(worktrees) == 0 {
			if listGlobal {
				fmt.Fprintln(out, "(no worktrees registered)")
			} else {
				fmt.Fprintln(out, "(no worktrees registered for this repo)")
			}
			return nil
		}

		writer := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		if listGlobal {
			fmt.Fprintln(writer, "REPO\tBRANCH\tSTATUS\tPATH")
			for _, w := range worktrees {
				fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
					w.RepoName, w.BranchName, w.Status, w.WorktreePath)
			}
		} else {
			fmt.Fprintln(writer, "BRANCH\tSTATUS\tPATH")
			for _, w := range worktrees {
				fmt.Fprintf(writer, "%s\t%s\t%s\n",
					w.BranchName, w.Status, w.WorktreePath)
			}
		}
		return writer.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listGlobal, "global", "g", false, "list worktrees across all repos")
	rootCmd.AddCommand(listCmd)
}
