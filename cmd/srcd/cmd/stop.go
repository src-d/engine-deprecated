package cmd

import (
	"github.com/spf13/cobra"
	"github.com/src-d/engine/components"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops all containers.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := components.Stop(); err != nil {
			fatal(err, "could not stop containers")
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
