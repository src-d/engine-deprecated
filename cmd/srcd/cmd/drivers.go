package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sirupsen/logrus"
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
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		// Might need to pull the image
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		drivers, err := c.ListDrivers(ctx, &api.ListDriversRequest{})
		if err != nil {
			logrus.Fatalf("could not list drivers: %v", err)
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

var parseDriversInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install language drivers.",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		for _, arg := range args {
			lang, version, err := parseDriverWithVersion(arg)
			if err != nil {
				logrus.Error(err)
				continue
			}

			logrus.WithFields(logrus.Fields{
				"language": lang,
				"version":  version,
			}).Info("installing driver")

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			_, err = c.InstallDriver(ctx, &api.VersionedDriver{Language: lang, Version: version})
			cancel()
			if err != nil {
				logrus.Errorf("unable to install version %s of %s driver: %s", version, lang, err)
			}
		}
	},
}

var parseDriversUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update installed language drivers.",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		for _, arg := range args {
			lang, version, err := parseDriverWithVersion(arg)
			if err != nil {
				logrus.Error(err)
				continue
			}

			logrus.WithFields(logrus.Fields{
				"language": lang,
				"version":  version,
			}).Info("updating driver")

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			_, err = c.UpdateDriver(ctx, &api.VersionedDriver{Language: lang, Version: version})
			cancel()
			if err != nil {
				logrus.Errorf("unable to update %s driver to version %s: %s", lang, version, err)
			}
		}
	},
}

var parseDriversRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove installed language drivers.",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		for _, lang := range args {
			logrus.Infof("removing %s driver", lang)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, err = c.RemoveDriver(ctx, &api.RemoveDriverRequest{Language: lang})
			cancel()
			if err != nil {
				logrus.Fatalf("unable to remove drivers: %s", err)
			}
		}
	},
}

func parseDriverWithVersion(arg string) (lang, version string, err error) {
	parts := strings.Split(arg, ":")
	lang = strings.ToLower(parts[0])
	switch len(parts) {
	case 1: // do nothing
	case 2:
		version = strings.ToLower(parts[1])
	default:
		return "", "", fmt.Errorf("invalid argument format: %s", arg)
	}

	if version == "" {
		version = "latest"
	}

	return
}

func init() {
	parseDriversCmd.AddCommand(parseDriversListCmd)
	parseDriversCmd.AddCommand(parseDriversInstallCmd)
	parseDriversCmd.AddCommand(parseDriversUpdateCmd)
	parseDriversCmd.AddCommand(parseDriversRemoveCmd)
}
