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
	"os/signal"

	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
	"github.com/src-d/engine/components"
	"github.com/src-d/engine/docker"
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
			query = strings.TrimSpace(args[0])
		}

		client, err := daemon.Client()
		if err != nil {
			fatal(err, "could not get daemon client")
		}

		startGitbaseWithClient(client)
		connReady := logAfterTimeoutWithSpinner("waiting for gitbase to be ready", 5*time.Second, 0)
		err = ensureConnReady(client)
		connReady()
		if err != nil {
			fatal(err, "could not connect to gitbase")
		}

		// Support piping
		// TODO(@smacker): not the most optimal solution
		// it would read all input into memory first and only then send to gitbase
		// it must be possible to pipe and running mysql-cli with -B flag
		// but it would change current client behaviour
		fi, _ := os.Stdin.Stat()
		if (fi.Mode() & os.ModeCharDevice) == 0 {
			b, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				fatal(err, "could not read input")
			}

			query = string(b)
		}

		resp, exit, err := runMysqlCli(context.Background(), query)
		if err != nil {
			fatal(err, "could not run mysql client")
		}
		defer resp.Close()
		defer stopMysqlClient()

		// in case of Ctrl-C or kill defer wouldn't work
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, os.Kill)
		go func() {
			<-ch
			stopMysqlClient()
		}()

		if query != "" {
			if _, err = io.Copy(os.Stdout, resp.Reader); err != nil {
				return err
			}

			cd := int(<-exit)
			os.Exit(cd)
			return nil
		}

		return attachStdio(resp)
	},
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

func startGitbaseWithClient(client api.EngineClient) {
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
		fatal(err, "could not start gitbase")
	}

	if err := docker.EnsureInstalled(components.MysqlCli.Image, components.MysqlCli.Version); err != nil {
		fatal(err, "could not install mysql client")
	}
}

func runMysqlCli(ctx context.Context, query string, opts ...docker.ConfigOption) (*types.HijackedResponse, chan int64, error) {
	cmd := []string{"mysql", "-h", components.Gitbase.Name}
	if query != "" {
		cmd = append(cmd, "-e", query)
	}

	config := &container.Config{
		Image: components.MysqlCli.ImageWithVersion(),
		Cmd:   cmd,
	}
	host := &container.HostConfig{}
	docker.ApplyOptions(config, host, opts...)

	return docker.Attach(context.Background(), config, host, components.MysqlCli.Name)
}

func attachStdio(resp *types.HijackedResponse) error {
	inputDone := make(chan error)
	outputDone := make(chan error)

	go func() {
		_, err := io.Copy(os.Stdout, resp.Reader)
		outputDone <- err
		resp.CloseWrite()
	}()

	go func() {
		_, err := io.Copy(resp.Conn, os.Stdin)

		if err := resp.CloseWrite(); err != nil {
			logrus.Debugf("Couldn't send EOF: %s", err)
		}

		inputDone <- err
	}()

	select {
	case err := <-outputDone:
		return err
	case err := <-inputDone:
		if err == nil {
			// Wait for output to complete streaming.
			return <-outputDone
		}

		return err
	}
}

func stopMysqlClient() {
	err := docker.RemoveContainer(components.MysqlCli.Name)
	if err != nil {
		logrus.Warnf("could not stop mysql client: %v", err)
	}
}

func init() {
	rootCmd.AddCommand(sqlCmd)
}
