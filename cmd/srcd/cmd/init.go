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

	"github.com/src-d/engine/cmd/srcd/config"
	"github.com/src-d/engine/cmd/srcd/daemon"

	"gopkg.in/src-d/go-log.v1"
)

// initCmd represents the init command
type initCmd struct {
	Command `name:"init" short-description:"Starts the daemon or restarts it if already running" long-description:"Starts the daemon or restarts it if already running"`

	Args struct {
		Workdir string `positional-arg-name:"workdir"`
	} `positional-args:"yes"`
}

func (c *initCmd) Execute(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments, expected only one path")
	}

	config.InitConfig(c.Config)

	var err error
	workdir := c.Args.Workdir

	workdir = strings.TrimSpace(workdir)
	if workdir == "" {
		workdir, err = os.Getwd()
	} else {
		workdir, err = filepath.Abs(workdir)
	}

	if err != nil {
		return humanizef(err, "could not get working directory")
	}

	info, err := os.Stat(workdir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("path '%s' is not a valid working directory", workdir)
	}

	err = daemon.Kill()
	if err != nil {
		return humanizef(err, "could not stop daemon")
	}

	log.Infof("starting daemon with working directory: %s", workdir)

	if err := daemon.Start(workdir); err != nil {
		return humanizef(err, "could not start daemon")
	}

	log.Infof("daemon started")
	return nil
}

func init() {
	rootCmd.AddCommand(&initCmd{})
}
