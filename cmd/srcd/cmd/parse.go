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

	api "github.com/src-d/engine/api"
	"github.com/src-d/engine/cmd/srcd/daemon"

	"gopkg.in/src-d/go-cli.v0"
	"gopkg.in/src-d/go-log.v1"
)

// parseCmd represents the parse command
type parseCmd struct {
	cli.PlainCommand `name:"parse" short-description:"Classify languages, parse files, and manage parsers" long-description:"Classify languages, parse files, and manage parsers"`
}

// parseUASTCmd represents the parse uast command
type parseUASTCmd struct {
	Command `name:"uast" short-description:"Parse and return the filtered UAST of the given file" long-description:"Parse and return the filtered UAST of the given file\n\nThis command parses the given file, automatically identifying the language\nunless the --lang flag is used. The resulting Universal Abstract Syntax Trees\n(UASTs) are filtered with the given --query XPath expression. By default it\nreturns UAST in semantic mode, it can be changed using --mode flag.\n\nThe remaining nodes are printed to standard output in JSON format."`

	Lang  string `short:"l" long:"lang" description:"avoid language detection, use this parser"`
	Query string `short:"q" long:"query" description:"XPath query applied to the parsed UASTs"`
	Mode  string `short:"m" long:"mode" choice:"semantic" choice:"annotated" choice:"native" default:"semantic" description:"UAST parsing mode"`

	Args struct {
		Path string `positional-arg-name:"file-path" required:"yes"`
	} `positional-args:"yes"`
}

func (cmd *parseUASTCmd) Execute(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments, expected only one path")
	}

	b, err := ioutil.ReadFile(cmd.Args.Path)
	if err != nil {
		return humanizef(err, "could not read %s", cmd.Args.Path)
	}

	c, err := daemon.Client()
	if err != nil {
		return humanizef(err, "could not get daemon client")
	}

	// First time it can be quite slow, as it may have to pull images.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	timeout := 3 * time.Second
	started := logAfterTimeout("this is taking a while, "+
		"if this is the first time you launch the parsing client, "+
		"it might take a few more minutes while we install all the required images",
		timeout)

	mode, err := parseModeArg(cmd.Mode)
	if err != nil {
		return err
	}

	lang := cmd.Lang
	var resp *api.ListDriversResponse

	if lang == "" {
		lang, err = parseLang(ctx, c, cmd.Args.Path, b)
		started()

		if err != nil {
			return humanizef(err, "cannot parse language")
		}

		log.Infof("detected language: %s", lang)
		resp, err = c.ListDrivers(ctx, &api.ListDriversRequest{})
	} else {
		resp, err = c.ListDrivers(ctx, &api.ListDriversRequest{})
		started()
	}

	if err != nil {
		return humanizef(err, "could not list drivers")
	}

	err = checkSupportedLanguage(resp.Drivers, lang)
	if err != nil {
		return err
	}

	stream, err := c.ParseWithLogs(ctx, &api.ParseRequest{
		Kind:    api.ParseRequest_UAST,
		Name:    cmd.Args.Path,
		Content: b,
		Lang:    lang,
		Query:   cmd.Query,
		Mode:    mode,
	})
	if err != nil {
		return humanizef(err, "%T", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return fmt.Errorf("stream closed unexpectedly")
		}

		if err != nil {
			return humanizef(err, "could not stream")
		}

		switch resp.Kind {
		case api.ParseResponse_FINAL:
			for _, node := range resp.Uast {
				fmt.Println(string(node))
			}

			return nil
		case api.ParseResponse_LOG:
			log.Debugf(resp.Log)
		}
	}
}

// parseLangCmd represents the parse lang command
type parseLangCmd struct {
	Command `name:"lang" short-description:"Identify the language of the given file" long-description:"Identify the language of the given file"`

	Args struct {
		Path string `positional-arg-name:"file-path" required:"yes"`
	} `positional-args:"yes"`
}

func (cmd *parseLangCmd) Execute(args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("too many arguments, expected only one path")
	}

	b, err := ioutil.ReadFile(cmd.Args.Path)
	if err != nil {
		return humanizef(err, "could not read %s", cmd.Args.Path)
	}

	c, err := daemon.Client()
	if err != nil {
		return humanizef(err, "could not get daemon client")
	}

	lang, err := parseLang(context.Background(), c, cmd.Args.Path, b)
	if err != nil {
		return humanizef(err, "cannot parse language")
	}

	fmt.Println(lang)

	return nil
}

func init() {
	c := rootCmd.AddCommand(&parseCmd{})
	c.AddCommand(&parseUASTCmd{})
	c.AddCommand(&parseLangCmd{})
	c.AddCommand(&parseDriverCmd{})
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
