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
	"strings"

	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"

	"gopkg.in/src-d/go-cli.v0"
	"gopkg.in/src-d/go-log.v1"
)

// componentsCmd represents the components command
type componentsCmd struct {
	cli.PlainCommand `name:"components" short-description:"Manage source{d} components and their installations" long-description:"Manage source{d} components and their installations"`
}

// componentsListCmd represents the components list command
type componentsListCmd struct {
	Command `name:"list" short-description:"List source{d} components" long-description:"List source{d} components"`

	All bool `short:"a" long:"all" description:"show all versions found"`
}

func (c *componentsListCmd) Execute(args []string) error {
	components.Daemon.RetrieveVersion()

	cmps, err := components.List(context.Background(), c.All)
	if err != nil {
		return humanizef(err, "could not list images")
	}

	t := NewTable("%s", "%s", "%v", "%v", "%v")
	t.Header("IMAGE", "INSTALLED", "RUNNING", "PORT", "CONTAINER NAME")
	for _, cmp := range cmps {
		t.Row(
			cmp.ImageWithVersion(),
			boolFmt(cmp.IsInstalled()),
			boolFmt(cmp.IsRunning()),
			publicPortsFmt(cmp.GetPorts()),
			cmp.Name,
		)
	}

	return t.Print(os.Stdout)
}

func boolFmt(b bool, err error) string {
	if err != nil {
		return "?"
	}
	if b {
		return "yes"
	}

	return "no"
}

func publicPortsFmt(ps []docker.Port, err error) string {
	if err != nil {
		return "?"
	}

	var publicPorts []string
	for _, p := range ps {
		if p.PublicPort != 0 {
			publicPorts = append(publicPorts, fmt.Sprintf("%d", p.PublicPort))
		}
	}

	return strings.Join(publicPorts, ",")
}

// componentsInstallCmd represents the components install command
type componentsInstallCmd struct {
	Command `name:"install" short-description:"Install source{d} component" long-description:"Install source{d} component"`

	Args struct {
		Components []string `positional-arg-name:"component(s)" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

func (c *componentsInstallCmd) Execute(args []string) error {
	cmps, err := components.List(context.Background(), false)
	if err != nil {
		return humanizef(err, "could not list images")
	}

	for _, arg := range c.Args.Components {
		var c *components.Component
		for _, cmp := range cmps {
			// We allow to match by container name or by image name
			if arg == cmp.Name || arg == cmp.Image {
				c = &cmp
				break
			}
		}

		if c == nil {
			names := make([]string, len(cmps))
			for i, cmp := range cmps {
				names[i] = cmp.Image
			}

			return fmt.Errorf("%s is not valid. Component must be one of [%s]", arg, strings.Join(names, ", "))
		}

		_, err = c.RetrieveVersion()
		if err != nil {
			return humanizef(err, "could not retrieve the latest compatible version for %s", c.Image)
		}

		installed, err := c.IsInstalled()
		if err != nil {
			return humanizef(err, "could not check if %s is installed", arg)
		}

		if installed {
			log.Infof("%s is already installed", arg)
			continue
		}

		log.Infof("installing %s", c.ImageWithVersion())

		err = c.Install()
		if err != nil {
			return humanizef(err, "could not install %s", arg)
		}
	}

	return nil
}

func init() {
	c := rootCmd.AddCommand(&componentsCmd{})
	c.AddCommand(&componentsListCmd{})
	c.AddCommand(&componentsInstallCmd{})
}
