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
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"
)

var parseCmd = &cobra.Command{
	Use:   "parse",
	Short: "Classify languages, parse files, and manage parsers",
}

var parseUASTCmd = &cobra.Command{
	Use:   "uast [file-path]",
	Short: "Parse and return the filtered UAST of the given file",
	Long: `Parse and return the filtered UAST of the given file

This command parses the given file, automatically identifying the language
unless the --lang flag is used. The resulting Universal Abstract Syntax Trees
(UASTs) are filtered with the given --query XPath expression. By default it
returns UAST in semantic mode, it can be changed using --mode flag.

The remaining nodes are printed to standard output in JSON format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("file-path is required")
		}

		if len(args) > 1 {
			return fmt.Errorf("too many arguments, expected only one path")
		}

		path := args[0]

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read %s: %v", path, err)
		}

		c, err := daemon.Client()
		if err != nil {
			return fmt.Errorf("could not get daemon client: %v", err)
		}

		// First time it can be quite slow, as it may have to pull images.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		timeout := 3 * time.Second
		started := logAfterTimeout("this is taking a while, "+
			"if this is the first time you launch the parsing client, "+
			"it might take a few more minutes while we install all the required images",
			timeout)

		flags := cmd.Flags()
		lang, _ := flags.GetString("lang")
		query, _ := flags.GetString("query")
		modeArg, _ := flags.GetString("mode")
		mode, err := parseModeArg(modeArg)
		if err != nil {
			return err
		}

		var resp *api.ListDriversResponse
		if lang == "" {
			lang, err = parseLang(ctx, c, path, b)
			started()

			if err != nil {
				return fmt.Errorf("cannot parse language: %v", err)
			}

			logrus.Infof("detected language: %s", lang)
			resp, err = c.ListDrivers(ctx, &api.ListDriversRequest{})
		} else {
			resp, err = c.ListDrivers(ctx, &api.ListDriversRequest{})
			started()
		}

		if err != nil {
			return fmt.Errorf("could not list drivers: %v", err)
		}

		err = checkSupportedLanguage(resp.Drivers, lang)
		if err != nil {
			return err
		}

		stream, err := c.ParseWithLogs(ctx, &api.ParseRequest{
			Kind:    api.ParseRequest_UAST,
			Name:    path,
			Content: b,
			Lang:    lang,
			Query:   query,
			Mode:    mode,
		})
		if err != nil {
			return fmt.Errorf("%T %v", err, err)
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return fmt.Errorf("stream closed unexpectedly")
			}

			if err != nil {
				return fmt.Errorf("could not stream: %v", err)
			}

			switch resp.Kind {
			case api.ParseResponse_FINAL:
				for _, node := range resp.Uast {
					fmt.Println(string(node))
				}

				return nil
			case api.ParseResponse_LOG:
				logrus.Debugf(resp.Log)
			}
		}
	},
}

var parseLangCmd = &cobra.Command{
	Use:   "lang [file-path]",
	Short: "Identify the language of the given file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf("file-path is required")
		}

		if len(args) > 1 {
			return fmt.Errorf("too many arguments, expected only one path")
		}

		path := args[0]
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("could not read %s: %v", path, err)
		}

		c, err := daemon.Client()
		if err != nil {
			return fmt.Errorf("could not get daemon client: %v", err)
		}

		lang, err := parseLang(context.Background(), c, path, b)
		if err != nil {
			return fmt.Errorf("cannot parse language: %v", err)
		}

		fmt.Println(lang)

		return nil
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
	parseUASTCmd.Flags().StringP("mode", "m", "semantic", "UAST parsing mode: semantic|annotated|native")
}

func parseModeArg(mode string) (api.ParseRequest_UastMode, error) {
	switch mode {
	case "semantic":
		return api.ParseRequest_SEMANTIC, nil
	case "annotated":
		return api.ParseRequest_ANNOTATED, nil
	case "native":
		return api.ParseRequest_NATIVE, nil
	default:
		return api.ParseRequest_SEMANTIC, fmt.Errorf(
			"incorrect UAST mode '%s'. Allowed values: semantic, annotated, native", mode)
	}
}

func parseLang(ctx context.Context, client api.EngineClient, path string, b []byte) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	res, err := client.Parse(ctx, &api.ParseRequest{
		Kind:    api.ParseRequest_LANG,
		Name:    path,
		Content: b,
	})

	if err != nil {
		return "", fmt.Errorf("server error: %v", err)
	}

	return res.Lang, nil
}

func checkSupportedLanguage(supportedDrivers []*api.ListDriversResponse_DriverInfo, desired string) error {
	var langs []string
	isSupported := false
	for _, driver := range supportedDrivers {
		langs = append(langs, driver.Lang)
		if driver.Lang == desired {
			isSupported = true
		}
	}

	if isSupported {
		return nil
	}

	supportedLangsMsg := "'" + strings.Join(langs, "', '") + "'"
	return fmt.Errorf("language '%s' is not supported, currently supported languages are: %s",
		desired, supportedLangsMsg)
}
