package cmd

import "github.com/spf13/cobra"

var offCmd = &cobra.Command{
	Use:   "off",
	Short: "Stop the doctree daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shutdownDaemon()
	},
}

func init() {
	rootCmd.AddCommand(offCmd)
}