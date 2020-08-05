/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init [chain-id] [rpc-addr]",
	Short: "initialize the config file for the deploy application",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
			fmt.Printf("creating config %s...\n", cfgPath)
			if err = writeConfig(cmd, &Config{
				ChainID: args[0],
				RPCAddr: args[1],
				Keyfile: "key.priv",
				Keypass: defaultPass,
			}); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("Config %s already exists", cfgPath)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
