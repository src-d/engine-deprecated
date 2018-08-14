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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	enry "gopkg.in/src-d/enry.v1"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("not implemented")
	},
}

func printLanguageForPath(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		log.Printf("could not read %s: %v", path, err)
		return
	}

	if !fi.IsDir() {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("could not read %s: %v", path, err)
			return
		}
		lang := enry.GetLanguage(path, b)
		if lang == "" {
			lang = "Unknown"
		}
		fmt.Printf("%s: %s\n", path, lang)
		return
	}

	err = filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsDir() || !f.Mode().IsRegular() {
			return nil
		}
		printLanguageForPath(path)
		return nil
	})

	if err != nil {
		log.Printf("could not visit %s: %v", path, err)
	}
}

var parseLangCmd = &cobra.Command{
	Use:   "lang",
	Short: "Identify the language of the given files.",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}
		printLanguageForPath(args[0])
	},
}

func init() {
	parseCmd.AddCommand(parseUASTCmd)
	parseCmd.AddCommand(parseLangCmd)
	rootCmd.AddCommand(parseCmd)

	parseUASTCmd.Flags().StringP("lang", "l", "", "avoid language detection, use this parser")
	parseUASTCmd.Flags().StringP("query", "q", "", "XPath query applied to the parsed UASTs")
}
