package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
)

var parseDriversListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed language drivers.",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := daemon.Client()
		if err != nil {
			fatal(err, "could not get daemon client")
		}

		// Might need to pull the image
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		drivers, err := c.ListDrivers(ctx, &api.ListDriversRequest{})
		if err != nil {
			fatal(err, "could not list drivers")
		}

		w := new(tabwriter.Writer)
		defer w.Flush()
		w.Init(os.Stdout, 0, 8, 5, '\t', 0)
		fmt.Fprintln(w, "LANGUAGE\tVERSION")
		fmt.Fprintln(w, "----------\t----------")
		for _, driver := range drivers.Drivers {
			fmt.Fprintf(w, "%s\t%s\n", driver.Lang, driver.Version)
		}
	},
}

func init() {
	parseDriversCmd.AddCommand(parseDriversListCmd)
}
