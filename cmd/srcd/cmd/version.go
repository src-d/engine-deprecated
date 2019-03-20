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

	"github.com/spf13/cobra"
	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
)

var version = ""

// SetVersion sets version for the command
func SetVersion(v string) {
	version = v
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("srcd cli version: %s\n", version)
		v, err := daemon.DockerVersion()
		if err != nil {
			return fatal(err, "could not get docker version")
		}

		fmt.Printf("docker version: %s\n", v)

		if ok, err := daemon.IsRunning(); err != nil {
			return fatal(err, "could not get srcd daemon version")
		} else if !ok {
			fmt.Printf("srcd daemon version: not running\n")
			return nil
		}

		client, err := daemon.Client()
		if err != nil {
			return fatal(err, "could not get daemon client")
		}

		res, err := client.Version(context.Background(), &api.VersionRequest{})
		if err != nil {
			return fatal(err, "could not get daemon version")
		}

		fmt.Printf("srcd daemon version: %s\n", res.Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
