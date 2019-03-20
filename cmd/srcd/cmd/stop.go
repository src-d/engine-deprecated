package cmd

import (
	"github.com/spf13/cobra"
	"github.com/src-d/engine/components"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops all containers.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := components.Stop(); err != nil {
			return fatal(err, "could not stop containers")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
