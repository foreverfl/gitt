package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the installed doctree version",
	RunE: func(cmd *cobra.Command, args []string) error {
		v := installedVersion()
		if v == "" {
			fmt.Println("unknown (not installed via install.sh)")
			return nil
		}
		fmt.Println(v)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}