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

	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
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
			if err := repl(); err != nil {
				log.Fatal(err)
			}
			return nil
		}

		if err := runQuery(query); err != nil {
			log.Fatal(err)
		}
		return nil
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
		// read until you get a trailing ';'.
		var lines []string
		for {
			line, err := rl.Readline()
			if err != nil {
				if err != io.EOF {
					log.Fatalf("could not read line: %v", err)
				}
				return nil
			}
			line = strings.TrimSpace(line)
			lines = append(lines, line)
			if strings.HasSuffix(line, ";") {
				rl.SetPrompt("gitbase> ")
				break
			}
			rl.SetPrompt("      -> ")
		}

		// drop the trailing semicolon and all extra blank spaces.
		statement := strings.Join(lines, "\n")
		statement = strings.TrimSpace(strings.TrimSuffix(statement, ";"))
		switch strings.ToLower(statement) {
		case "exit", "quit":
			return nil
		case "":
		default:
			if err := runQuery(statement); err != nil {
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
	ctx, cancel := context.WithTimeout(context.Background(), 1440*time.Minute)
	defer cancel()

	stream, err := c.SQL(ctx, &api.SQLRequest{Query: query})
	if err != nil {
		// TODO(erizocosmico): extract the actual error from the transport
		return err
	}

	resp, err := stream.Recv()
	if err != nil {
		return err
	}

	writer := tablewriter.NewWriter(os.Stdout)
	// reflow is very expensive it slows downs rendering of source code dramatically
	// and also "breaks" code
	writer.SetReflowDuringAutoWrap(false)
	writer.SetHeader(resp.Row.Cell)

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			writer.Render()
			return nil
		}
		if err != nil {
			return err
		}
		writer.Append(resp.Row.Cell)
	}
}

func init() {
	rootCmd.AddCommand(sqlCmd)
}
