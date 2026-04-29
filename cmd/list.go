package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/foreverfl/gitt/internal/daemon"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all gitt-managed worktrees from the daemon's database",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireDaemon(); err != nil {
			return err
		}
		worktrees, err := daemon.ListWorktrees()
		if err != nil {
			return err
		}
		if len(worktrees) == 0 {
			fmt.Println("(no worktrees registered)")
			return nil
		}

		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(writer, "REPO\tBRANCH\tSTATUS\tPATH")
		for _, w := range worktrees {
			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n",
				w.RepoName, w.BranchName, w.Status, w.WorktreePath)
		}
		return writer.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
