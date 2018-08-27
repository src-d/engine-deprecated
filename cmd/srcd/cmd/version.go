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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
)

const version = "0.0.1"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("srcd cli version: %s\n", version)
		if v, err := daemon.DockerVersion(); err != nil {
			fmt.Printf("could not get docker version: %s\n", err)
		} else {
			fmt.Printf("docker version: %s\n", v)
		}

		if ok, err := daemon.IsRunning(); err != nil {
			fmt.Printf("could not get srcd daemon version: %s\n", err)
			return
		} else if !ok {
			fmt.Printf("srcd daemon version: not running\n")
			return
		}

		client, err := daemon.Client()
		if err != nil {
			logrus.Fatal(err)
		}
		res, err := client.Version(context.Background(), &api.VersionRequest{})
		if err != nil {
			logrus.Fatal(err)
		}
		fmt.Printf("srcd daemon version: %s\n", res.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
