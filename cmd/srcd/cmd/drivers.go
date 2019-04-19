package cmd

import (
	"context"
	"os"
	"time"

	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
)

// parseDriverCmd represents the parse drivers command
type parseDriverCmd struct {
	Command `name:"drivers" short-description:"List installed language drivers" long-description:"List installed language drivers"`
}

func (cmd *parseDriverCmd) Execute(args []string) error {
	c, err := daemon.Client()
	if err != nil {
		return humanizef(err, "could not get daemon client")
	}

	// Might need to pull the image
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	drivers, err := c.ListDrivers(ctx, &api.ListDriversRequest{})
	if err != nil {
		return humanizef(err, "could not list drivers")
	}

	t := NewTable("%s", "%s")
	t.Header("LANGUAGE", "VERSION")
	for _, driver := range drivers.Drivers {
		t.Row(driver.Lang, driver.Version)
	}

	return t.Print(os.Stdout)
}
