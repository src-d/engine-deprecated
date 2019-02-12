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
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
	"github.com/src-d/engine/components"
)

// sqlCmd represents the sql command
var sqlCmd = &cobra.Command{
	Use:   "sql [query]",
	Short: "Run a SQL query over the analyzed repositories.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 1 {
			return fmt.Errorf("too many arguments, expected only one query or nothing")
		}

		var query string
		if len(args) == 1 {
			query = args[0]
		}

		client, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		started := logAfterTimeout("this is taking a while, " +
			"if this is the first time you launch sql client, " +
			"it might take a few more minutes while we install all the required images")

		// Might have to pull some images
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		_, err = client.StartComponent(ctx, &api.StartComponentRequest{
			Name: components.Gitbase.Name,
		})
		close(started)
		cancel()
		if err != nil {
			logrus.Fatalf("could not start gitbase: %v", err)
		}

		connReady := logAfterTimeout("waiting for gitbase to be ready")
		if err := ensureConnReady(client); err != nil {
			logrus.Fatalf("could not connect to gitbase: %v", err)
		}
		close(connReady)

		if strings.TrimSpace(query) == "" {
			if err := repl(client); err != nil {
				log.Fatal(err)
			}
			return nil
		}

		if err := runQuery(client, query); err != nil {
			log.Fatal(err)
		}
		return nil
	},
}

func repl(client api.EngineClient) error {
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
			if err := runQuery(client, statement); err != nil {
				fmt.Println(err)
			}
		}
	}
}

func ensureConnReady(client api.EngineClient) error {
	ctx := context.Background()

	done := make(chan error)
	globalTimeout := 5 * time.Minute
	go func(ctx context.Context) {
		queryTimeout := 1 * time.Second
		sleep := 1 * time.Second
		for {
			err := pingDB(ctx, client, queryTimeout)
			if err == nil {
				break
			}

			time.Sleep(sleep)
		}

		done <- nil
	}(ctx)

	ctx, cancel := context.WithTimeout(ctx, globalTimeout)
	defer cancel()

	select {
	case <-ctx.Done():
		return fmt.Errorf("global timeout of %v exceeded", globalTimeout)
	case <-done:
		return nil
	}
}

func pingDB(ctx context.Context, client api.EngineClient, queryTimeoutSeconds time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeoutSeconds)
	defer cancel()

	done := make(chan error)
	go func(ctx context.Context, done chan error) {
		stream, err := client.SQL(ctx, &api.SQLRequest{Query: "SELECT 1"})
		if err != nil {
			done <- err
		}

		_, err = stream.Recv()
		if err != nil {
			done <- err
		}

		done <- nil
	}(ctx, done)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func runQuery(client api.EngineClient, query string) error {
	// Might have to pull some images
	ctx, cancel := context.WithTimeout(context.Background(), 1440*time.Minute)
	defer cancel()

	stream, err := client.SQL(ctx, &api.SQLRequest{Query: query})
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

	writer.SetHeader(toStr(resp.Row.Cell))

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			writer.Render()
			return nil
		}
		if err != nil {
			return err
		}
		writer.Append(toStr(resp.Row.Cell))
	}
}

func toStr(bytes [][]byte) []string {
	strings := make([]string, len(bytes))
	for i, v := range bytes {
		strings[i] = string(v)
	}

	return strings
}

func init() {
	rootCmd.AddCommand(sqlCmd)
}
