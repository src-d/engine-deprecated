// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
	"github.com/src-d/engine/components"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start web interfaces for source{d} tools",
}

var webSQLCmd = &cobra.Command{
	Use:   "sql",
	Short: "Start gitbase web client",
	Run:   startWebComponent(components.GitbaseWeb.Name, "gitbase web client"),
}

var webParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Start bblfsh web client",
	Run:   startWebComponent(components.BblfshWeb.Name, "bblfsh web client"),
}

func startWebComponent(name, desc string) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		started := logAfterTimeoutWithServerLogs("this is taking a while, if this is the first time you launch this web client, it might take a few more minutes while we install all the required images",
			3*time.Second)

		// Might have to pull some images
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

		res, err := c.StartComponent(ctx, &api.StartComponentRequest{
			Name: name,
		})
		started()

		if err != nil {
			cancel()
			logrus.Fatalf("could not start %s: %v", desc, err)
		}
		cancel()

		fmt.Printf("Go to http://localhost:%d for the %s. Press Ctrl-C to stop it.\n", res.Port, desc)
		_ = browser.OpenURL(fmt.Sprintf("http://localhost:%d", res.Port))

		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, os.Kill)

		<-ch

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		_, err = c.StopComponent(ctx, &api.StopComponentRequest{Name: name})
		if err != nil {
			cancel()
			logrus.Fatalf("could not stop %s: %v", desc, err)
		}

		close(ch)
	}
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.AddCommand(webSQLCmd)
	webCmd.AddCommand(webParseCmd)
}
