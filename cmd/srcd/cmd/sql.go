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
	"time"

	"github.com/chzyer/readline"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/cmd/srcd/daemon"
)

// sqlCmd represents the sql command
var sqlCmd = &cobra.Command{
	Use:   "sql",
	Short: "Run a SQL query over the analyzed repositories.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("two many arguments, expected only one query or nothing")
		}

		var query string
		if len(args) == 1 {
			query = args[0]
		}

		if strings.TrimSpace(query) == "" {
			return repl()
		}

		return runQuery(query)
	},
}

func repl() error {
	rl, err := readline.NewEx(&readline.Config{
		Prompt: "gitbase> ",
		Stdin:  os.Stdin,
		Stderr: os.Stderr,
		Stdout: os.Stdout,
	})
	if err != nil {
		return err
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			return nil
		}

		switch clean(line) {
		case "exit", "quit":
			return nil
		default:
			if err := runQuery(line); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func runQuery(query string) error {
	c, err := daemon.Client()
	if err != nil {
		return fmt.Errorf("could not get daemon client: %v", err)
	}

	// Might have to pull some images
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	res, err := c.SQL(ctx, &api.SQLRequest{Query: query})
	if err != nil {
		// TODO(erizocosmico): extract the actual error from the transport
		return err
	}

	writer := tablewriter.NewWriter(os.Stdout)
	writer.SetHeader(res.Header.Cell)

	for _, row := range res.Rows {
		writer.Append(row.Cell)
	}

	writer.Render()
	return nil
}

func clean(line string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(line)), ";")
}

func init() {
	rootCmd.AddCommand(sqlCmd)
}
