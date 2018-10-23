package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/engine/components"
)

var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Stops and removes all containers and volumes used by engine.",
	Run: func(cmd *cobra.Command, args []string) {
		withImages, _ := cmd.Flags().GetBool("with-images")

		if err := components.Purge(withImages); err != nil {
			logrus.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(killCmd)

	killCmd.Flags().Bool("with-images", false, "remove docker images")
}
