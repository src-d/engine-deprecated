package cmd

import (
	"github.com/src-d/engine/cmd/srcd/daemon"
	"github.com/src-d/engine/components"
)

// pruneCmd represents the sql command
type pruneCmd struct {
	Command `name:"prune" short-description:"Removes all resources used by engine" long-description:"Removes all resources used by engine"`

	WithImages bool `long:"with-images" description:"remove docker images"`
}

func (c *pruneCmd) Execute(args []string) error {
	if err := components.Prune(c.WithImages); err != nil {
		return humanizef(err, "could not prune components")
	}

	if err := daemon.CleanUp(); err != nil {
		return humanizef(err, "could not clean up")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(&pruneCmd{})
}
