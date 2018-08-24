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
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	api "github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/cmd/srcd/daemon"
	"gopkg.in/bblfsh/sdk.v1/uast"
)

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Classify languages, parse files, and manage parsers",
}

var parseUASTCmd = &cobra.Command{
	Use:   "uast",
	Short: "Parse and return the filtered UAST of the given file(s)",
	Long: `Parse and return the filtered UAST of the given file(s)

This command parses the given files, automatically identifying the language
unless the --lang flag is used. The resulting Universal Abstract Syntax Trees
(UASTs) are filtered with the given --query XPath expression.

The remaining nodes are printed to standard output in JSON format.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}
		if len(args) > 1 {
			logrus.Warnf("only taking into account the first file; ignoring the rest")
		}
		path := args[0]

		b, err := ioutil.ReadFile(path)
		if err != nil {
			logrus.Fatalf("could not read %s: %v", path, err)
		}

		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		// First time it can be quite slow, as it may have to pull images.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		time.AfterFunc(time.Second, func() {
			logrus.Info("installing drivers for the first time, it might take a couple more seconds")
		})

		flags := cmd.Flags()
		lang, _ := flags.GetString("lang")
		query, _ := flags.GetString("query")
		stream, err := c.ParseWithLogs(ctx, &api.ParseRequest{
			Kind:    api.ParseRequest_UAST,
			Name:    path,
			Content: b,
			Lang:    lang,
			Query:   query,
		})
		if err != nil {
			logrus.Fatalf("%T %v", err, err)
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				logrus.Fatalf("stream closed unexpectedly")
			}
			if err != nil {
				logrus.Fatalf("could not stream: %v", err)
			}
			switch resp.Kind {
			case api.ParseResponse_FINAL:
				for _, b := range resp.Uast {
					var node uast.Node
					logrus.Infof("detected language: %s", resp.Lang)
					if err := node.Unmarshal(b); err != nil {
						logrus.Fatalf("could not unmarshal UAST: %v", err)
					}
					fmt.Println(&node)
				}
				return
			case api.ParseResponse_LOG:
				logrus.Debugf(resp.Log)
			}
		}
	},
}

var parseLangCmd = &cobra.Command{
	Use:   "lang",
	Short: "Identify the language of the given files.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}
		if len(args) > 1 {
			logrus.Warnf("only taking into account the first file; ignoring the rest")
		}
		path := args[0]
		b, err := ioutil.ReadFile(path)
		if err != nil {
			logrus.Fatalf("could not read %s: %v", path, err)
		}

		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		res, err := c.Parse(ctx, &api.ParseRequest{
			Kind:    api.ParseRequest_LANG,
			Name:    path,
			Content: b,
		})
		if err != nil {
			logrus.Fatalf("server error: %v", err)
		}
		fmt.Println(res.Lang)
	},
}

var parseDriversCmd = &cobra.Command{
	Use:   "drivers",
	Short: "Manage language drivers.",
}

func init() {
	rootCmd.AddCommand(parseCmd)
	parseCmd.AddCommand(parseUASTCmd)
	parseCmd.AddCommand(parseLangCmd)
	parseCmd.AddCommand(parseDriversCmd)

	parseUASTCmd.Flags().StringP("lang", "l", "", "avoid language detection, use this parser")
	parseUASTCmd.Flags().StringP("query", "q", "", "XPath query applied to the parsed UASTs")
}
