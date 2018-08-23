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

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/cmd/srcd/daemon"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start web interfaces for source{d} tools",
}

var webSQLCmd = &cobra.Command{
	Use:   "sql",
	Short: "Start gitbase web",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		// Might have to pull some images
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

		port, _ := cmd.Flags().GetUint("port")
		_, err = c.StartGitbaseWeb(ctx, &api.StartGitbaseWebRequest{Port: int32(port)})
		if err != nil {
			cancel()
			logrus.Fatalf("could not start gitbase web at port %d: %v", port, err)
		}
		cancel()

		fmt.Printf("Go to http://localhost:%d for the SQL web client. Press Ctrl-C to stop it.\n", port)
		_ = browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))

		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, os.Kill)

		<-ch
		close(ch)

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		_, err = c.StopGitbaseWeb(ctx, &api.StopGitbaseWebRequest{})
		if err != nil {
			cancel()
			logrus.Fatalf("could not stop gitbase web at port %d: %v", port, err)
		}
	},
}

var webParseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Start bblfsh web",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		// Might have to pull some images
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

		port, _ := cmd.Flags().GetUint("port")
		_, err = c.StartBblfshWeb(ctx, &api.StartBblfshWebRequest{Port: int32(port)})
		if err != nil {
			cancel()
			logrus.Fatalf("could not start bblfsh web at port %d: %v", port, err)
		}
		cancel()

		fmt.Printf("Go to http://localhost:%d for the bblfsh web client. Press Ctrl-C to stop it.\n", port)
		_ = browser.OpenURL(fmt.Sprintf("http://localhost:%d", port))

		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, os.Kill)

		<-ch
		close(ch)

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		_, err = c.StopBblfshWeb(ctx, &api.StopBblfshWebRequest{})
		if err != nil {
			cancel()
			logrus.Fatalf("could not stop bblfsh web at port %d: %v", port, err)
		}
	},
}

func init() {
	rootCmd.AddCommand(webCmd)
	webCmd.AddCommand(webSQLCmd)
	webCmd.AddCommand(webParseCmd)

	webSQLCmd.Flags().UintP("port", "p", 8080, "port of the service")
	webParseCmd.Flags().UintP("port", "p", 8081, "port of the service")
}
