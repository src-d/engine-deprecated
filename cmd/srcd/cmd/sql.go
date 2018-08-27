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
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/src-d/engine-cli/api"
	"github.com/src-d/engine-cli/cmd/srcd/daemon"
)

// sqlCmd represents the sql command
var sqlCmd = &cobra.Command{
	Use:   "sql",
	Short: "Run a SQL query over the analyzed repositories.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("missing query argument")
		} else if len(args) > 1 {
			return fmt.Errorf("two many arguments, expected only one query")
		}
		query := args[0]

		c, err := daemon.Client()
		if err != nil {
			logrus.Fatalf("could not get daemon client: %v", err)
		}

		// Might have to pull some images
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		res, err := c.SQL(ctx, &api.SQLRequest{Query: query})
		if err != nil {
			logrus.Fatalf("server error: %v", err)
		}

		writer := tablewriter.NewWriter(os.Stdout)
		writer.SetHeader(res.Header.Cell)

		for _, row := range res.Rows {
			writer.Append(row.Cell)
		}

		writer.Render()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sqlCmd)
}
