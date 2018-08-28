package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/engine/components"
)

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Stops and removes all containers, volumes and docker images used by engine.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := components.Purge(); err != nil {
			logrus.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}
