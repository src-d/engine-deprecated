package cmd

import (
	"github.com/src-d/engine/components"
)

// stopCmd represents the stop command
type stopCmd struct {
	Command `name:"stop" short-description:"Stops all containers" long-description:"Stops all containers"`
}

func (c *stopCmd) Execute(args []string) error {
	if err := components.Stop(); err != nil {
		return humanizef(err, "could not stop containers")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(&stopCmd{})
}
