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
	"io/ioutil"

	"os"
	"strings"
	"time"

	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
)

// sqlCmd represents the sql command

type sqlCmd struct {
	Command `name:"sql" short-description:"Run a SQL query over the analyzed repositories" long-description:"Run a SQL query over the analyzed repositories"`

	Args struct {
		Query string `positional-arg-name:"query"`
	} `positional-args:"yes"`
}

func (c *sqlCmd) Execute(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments, expected only one query or nothing")
	}

	client, err := daemon.Client()
	if err != nil {
		return humanizef(err, "could not get daemon client")
	}

	if err := startGitbaseWithClient(client); err != nil {
		return err
	}

	connReady := logAfterTimeoutWithSpinner("waiting for gitbase to be ready", 5*time.Second, 0)
	err = ensureConnReady(client)
	connReady()
	if err != nil {
		return humanizef(err, "could not connect to gitbase")
	}

	var query string
	if c.Args.Query != "" {
		query = strings.TrimSpace(c.Args.Query)
	} else {
		// Support piping
		// TODO(@smacker): not the most optimal solution
		// it would read all input into memory first and only then send to gitbase
		// it must be possible to pipe and running mysql-cli with -B flag
		// but it would change current client behaviour
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			b, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				return humanizef(err, "could not read input")
			}

			query = string(b)
		}
	}

	if err = runMysqlCli(context.Background(), query); err != nil {
		return err
	}

	return nil
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

func startGitbaseWithClient(client api.EngineClient) error {
	started := logAfterTimeoutWithServerLogs("this is taking a while, "+
		"if this is the first time you launch sql client, "+
		"it might take a few more minutes while we install all the required images",
		5*time.Second)
	defer started()

	// Download & run dependencies
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	_, err := client.StartComponent(ctx, &api.StartComponentRequest{
		Name: components.Gitbase.Name,
	})
	if err != nil {
		return humanizef(err, "could not start gitbase")
	}

	return nil
}

func runMysqlCli(ctx context.Context, query string) error {
	interactive := true
	cmd := []string{"mysql"}
	if query != "" {
		cmd = append(cmd, "-t", "-e", query)
		interactive = false
	}

	insResp, err := docker.ExecAndAttach(context.Background(), interactive, components.Gitbase.Name, cmd...)

	if err != nil {
		return humanizef(err, "a problem occurred while trying to run mysql client")
	}

	if insResp.Running {
		return fmt.Errorf("MySQL cli is still running")
	}

	if insResp.ExitCode != 0 {
		return fmt.Errorf("MySQL cli returned with exit code %d", insResp.ExitCode)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(&sqlCmd{})
}
