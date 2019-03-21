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
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

// componentsCmd represents the components command
var componentsCmd = &cobra.Command{
	Use:   "components",
	Short: "Manage source{d} components and their installations",
}

// componentsListCmd represents the components list command
var componentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List source{d} components",
	RunE: func(cmd *cobra.Command, args []string) error {
		allVersions, _ := cmd.Flags().GetBool("all")

		components.Daemon.RetrieveVersion()

		cmps, err := components.List(context.Background(), allVersions)
		if err != nil {
			return fmt.Errorf("could not list images: %v", err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
		fmt.Fprintf(w, "IMAGE\tINSTALLED\tRUNNING\tPORT\tCONTAINER NAME\n")

		for _, cmp := range cmps {
			fmt.Fprintf(w, "%s\t%s\t%v\t%v\t%v\n",
				cmp.ImageWithVersion(),
				boolFmt(cmp.IsInstalled()),
				boolFmt(cmp.IsRunning()),
				publicPortsFmt(cmp.GetPorts()),
				cmp.Name,
			)
		}

		w.Flush()

		return nil
	},
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

// componentsCmd represents the components install command
var componentsInstallCmd = &cobra.Command{
	Use:   "install [component]",
	Short: "Install source{d} component",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmps, err := components.List(context.Background(), false)
		if err != nil {
			return fmt.Errorf("could not list images: %s", err)
		}

		for _, arg := range args {
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
				return fmt.Errorf("could not retrieve the latest compatible version for %s: %s", c.Image, err)

			}

			installed, err := c.IsInstalled()
			if err != nil {
				return fmt.Errorf("could not check if %s is installed: %s", arg, err)
			}

			if installed {
				log.Printf("%s is already installed", arg)
				continue
			}

			log.Printf("installing %s", c.ImageWithVersion())

			err = c.Install()
			if err != nil {
				return fmt.Errorf("could not install %s: %s", arg, err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(componentsCmd)
	componentsCmd.AddCommand(componentsListCmd)
	componentsCmd.AddCommand(componentsInstallCmd)

	componentsListCmd.Flags().BoolP("all", "a", false, "show all versions found")
}
