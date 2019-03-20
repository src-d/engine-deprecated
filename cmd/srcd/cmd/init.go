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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/engine/cmd/srcd/daemon"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [workdir]",
	Short: "Starts the daemon or restarts it if already running.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		var workdir string

		if len(args) > 0 {
			workdir = args[0]
		}

		workdir = strings.TrimSpace(workdir)
		if workdir == "" {
			workdir, err = os.Getwd()
		} else {
			workdir, err = filepath.Abs(workdir)
		}

		if err != nil {
			return fatal(err, "could not get working directory")
		}

		info, err := os.Stat(workdir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("path %q is not a valid working directory", workdir)
		}

		err = daemon.Kill()
		if err != nil {
			return fatal(err, "could not stop daemon")
		}

		logrus.Infof("starting daemon with working directory: %s", workdir)

		if err := daemon.Start(workdir); err != nil {
			return fatal(err, "could not start daemon")
		}

		logrus.Info("daemon started")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
