package cmd

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type helpEntry struct {
	usage          string
	description    string
	requiresDaemon bool
}

// helpCommands mirrors the Commands table in README.md. Keep this list in sync
// when adding, removing, or renaming a subcommand.
var helpCommands = []helpEntry{
	{usage: "on", description: "Start the daemon (~/.gitt/gitt.sock, ~/.gitt/gitt.db)"},
	{usage: "off", description: "Stop the daemon"},
	{usage: "clone <url> [dir]", description: "Clone a repo into gitt's bare layout (<dir>/.bare + .worktrees/<default-branch>)"},
	{usage: "add <branch>", description: "Create a worktree at <repo>/.worktrees/<branch>", requiresDaemon: true},
	{usage: "remove <branch>", description: "Remove the worktree folder for <branch> (git worktree remove)", requiresDaemon: true},
	{usage: "rename <old> <new>", description: "Rename a branch and its worktree folder together", requiresDaemon: true},
	{usage: "status", description: "Show the current worktree's repo, branch, path, and state"},
	{usage: "vscode", description: "Write <repo>/<repo>.code-workspace with one folder per worktree. Runs from any path in the repo; output always lands at the main repo root. Open with `code <repo>/<repo>.code-workspace` — don't open `.worktrees` directly", requiresDaemon: true},
	{usage: "sqlite", description: "Run a SQLite self-test against the daemon's database", requiresDaemon: true},
	{usage: "update", description: "Fetch and install the latest gitt release"},
	{usage: "version", description: "Print the installed gitt version"},
	{usage: "logo", description: "Print the gitt logo art in a sky-blue box"},
	{usage: "uninstall", description: "Stop the daemon, remove ~/.gitt/, remove the binary"},
	{usage: "help", description: "Show this help"},
}

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show gitt help",
	Run: func(cmd *cobra.Command, args []string) {
		printHelp(os.Stdout)
	},
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, "gitt — git worktree + docker compose orchestrator")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  gitt <command> [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")

	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, entry := range helpCommands {
		description := entry.description
		if entry.requiresDaemon {
			description += "  [requires daemon]"
		}
		fmt.Fprintf(tw, "  %s\t%s\n", entry.usage, description)
	}
	tw.Flush()

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	fmt.Fprintln(out, "  -y, --yes   Skip confirmation prompts")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run `gitt <command> --help` for command-specific details.")
}

func init() {
	rootCmd.SetHelpCommand(helpCmd)
}
