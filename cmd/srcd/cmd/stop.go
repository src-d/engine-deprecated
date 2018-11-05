package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/engine/components"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops all containers.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := components.Stop(); err != nil {
			logrus.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
