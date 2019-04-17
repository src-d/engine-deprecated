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
	"os/signal"
	"time"

	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
	"github.com/src-d/engine/components"

	"github.com/pkg/browser"
	"gopkg.in/src-d/go-cli.v0"
)

// webCmd represents the web command
type webCmd struct {
	cli.PlainCommand `name:"web" short-description:"Start web interfaces for source{d} tools" long-description:"Start web interfaces for source{d} tools"`
}

// webSQLCmd represents the web sql command
type webSQLCmd struct {
	Command `name:"sql" short-description:"Start gitbase web client" long-description:"Start gitbase web client"`
}

func (c *webSQLCmd) Execute(args []string) error {
	return startWebComponent(components.GitbaseWeb.Name, "gitbase web client", args)
}

// webParseCmd represents the web parse command
type webParseCmd struct {
	Command `name:"parse" short-description:"Start bblfsh web client" long-description:"Start bblfsh web client"`
}

func (c *webParseCmd) Execute(args []string) error {
	return startWebComponent(components.BblfshWeb.Name, "bblfsh web client", args)
}

func startWebComponent(name, desc string, args []string) error {
	c, err := daemon.Client()
	if err != nil {
		return humanizef(err, "could not get daemon client")
	}

	// in case of gitbase-web we need to run gitbase first and make sure it started
	if name == components.GitbaseWeb.Name {
		timeout := 3 * time.Second
		started := logAfterTimeoutWithServerLogs("this is taking a while, "+
			"it might take a few more minutes while we install all the required images",
			timeout)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		_, err = c.StartComponent(ctx, &api.StartComponentRequest{
			Name: components.Gitbase.Name,
		})
		started()
		cancel()

		if err != nil {
			return humanizef(err, "could not start gitbase")
		}

		connReady := logAfterTimeoutWithSpinner("waiting for gitbase to be ready", timeout, 0)
		err = ensureConnReady(c)
		connReady()
		if err != nil {
			return humanizef(err, "could not connect to gitbase")
		}
	}

	started := logAfterTimeoutWithServerLogs("this is taking a while, if this is the first time you launch this web client, it might take a few more minutes while we install all the required images",
		3*time.Second)

	// Might have to pull some images
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

	res, err := c.StartComponent(ctx, &api.StartComponentRequest{
		Name: name,
	})
	started()

	if err != nil {
		cancel()
		return humanizef(err, "could not start %s", desc)
	}
	cancel()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, os.Kill)

	fmt.Printf("Go to http://localhost:%d for the %s. Press Ctrl-C to stop it.\n", res.Port, desc)
	_ = browser.OpenURL(fmt.Sprintf("http://localhost:%d", res.Port))

	<-ch

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	_, err = c.StopComponent(ctx, &api.StopComponentRequest{Name: name})
	if err != nil {
		cancel()
		return humanizef(err, "could not stop %s", desc)
	}

	close(ch)
	return nil
}

func init() {
	c := rootCmd.AddCommand(&webCmd{})
	c.AddCommand(&webSQLCmd{})
	c.AddCommand(&webParseCmd{})
}
