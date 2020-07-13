// Copyright © 2020 NAME HERE <EMAIL ADDRESS>
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
	"path"

	"github.com/ovrclk/akash/cmd/common"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

// startCmd represents the watch command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Listen to the chain and configuration directory and print those events",
	RunE: func(cmd *cobra.Command, args []string) error {
		return common.RunForever(func(ctx context.Context) error {
			dirs := []string{homePath, path.Join(homePath, "deployments")}
			return ChainAndFSEmitter(dirs)(ctx, PrintHandler)
		})
	},
}
