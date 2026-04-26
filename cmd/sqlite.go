package cmd

import (
	"errors"
	"fmt"

	"github.com/foreverfl/doctree/internal/daemon"
	"github.com/foreverfl/doctree/internal/paths"
	"github.com/spf13/cobra"
)

var sqliteCmd = &cobra.Command{
	Use:   "sqlite",
	Short: "Run a SQLite self-test against the daemon's database",
	Long: "Asks the running doctree daemon to create a scratch table, insert a\n" +
		"row, read it back, and drop the table. Useful for confirming the\n" +
		"daemon's database connection is healthy.\n\n" +
		"Requires `doctree on` to be running.",
	RunE: func(cmd *cobra.Command, args []string) error {
		sockpath, err := paths.SockPath()
		if err != nil {
			return err
		}
		response, err := daemon.Call(sockpath, daemon.Request{Op: daemon.OpSqliteTest})
		if err != nil {
			if errors.Is(err, daemon.ErrNotRunning) {
				return fmt.Errorf("doctree daemon not running. start it with `doctree on`")
			}
			return err
		}
		if !response.OK {
			return fmt.Errorf("sqlite test failed: %s", response.Error)
		}
		message, _ := response.Data["message"].(string)
		fmt.Println(message)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sqliteCmd)
}
