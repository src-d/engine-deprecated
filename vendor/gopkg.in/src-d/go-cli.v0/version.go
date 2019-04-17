package cli

import (
	"fmt"
)

// VersionCommand defines he default version command. Most if the time, it
// should not be used directly, since it will be added by default to the App.
type VersionCommand struct {
	PlainCommand `name:"version" short-description:"print version" long-description:"print version and exit"`
	Name         string
	Version      string
	Build        string
}

// Execute runs the version command.
func (c VersionCommand) Execute(args []string) error {
	fmt.Printf("%s version %s build %s\n", c.Name, c.Version, c.Build)
	return nil
}
