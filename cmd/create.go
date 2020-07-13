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
	"fmt"
	"io/ioutil"
	"path"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ovrclk/akash/sdl"
	dcli "github.com/ovrclk/akash/x/deployment/client/cli"
	dtypes "github.com/ovrclk/akash/x/deployment/types"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func init() {
	rootCmd.AddCommand(createCmd())
}

// createCmd represents the create command
func createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [sdl-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Create a deployment to be managed by the deploy application",
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := ioutil.ReadFile(args[0])
			if err != nil {
				return err
			}

			sdl, err := sdl.Read(file)
			if err != nil {
				return err
			}

			groups, err := sdl.DeploymentGroups()
			if err != nil {
				return err
			}

			id, err := dcli.DeploymentIDFromFlags(cmd.Flags(), config.GetAccAddress().String())
			if err != nil {
				return err
			}

			// Default DSeq to the current block height
			if id.DSeq == 0 {
				if id.DSeq, err = dcli.CurrentBlockHeight(config.CLICtx(config.NewTMClient())); err != nil {
					return err
				}
			}

			ctx, _ := context.WithCancel(context.Background())
			group, _ := errgroup.WithContext(ctx)

			group.Go(func() error {
				if err = WatchForChainAndFSEvents(ctx, PrintBusEvents, SendManifestHander); err != nil {
					fmt.Println("in watch for chains and fs events")
				}
				return err
			})

			group.Go(func() error {
				if err = txCreateDeployment(groups, id); err != nil {
					fmt.Println("in tx create deployemtn")
				}
				return err
			})

			group.Go(func() error {
				if err = createDeploymentFileInArchive(file, id); err != nil {
					fmt.Println("in create archive")
				}
				return err
			})

			// TODO: One more goroutine to wait for the site to be available and call cancel

			return group.Wait()
		},
	}
	dcli.AddDeploymentIDFlags(cmd.Flags())
	return cmd
}

func createDeploymentFileInArchive(file []byte, id dtypes.DeploymentID) error {
	fileName := fmt.Sprintf("%s.%d.yaml", id.Owner, id.DSeq)
	return ioutil.WriteFile(path.Join(homePath, "deployments", fileName), file, 666)
}

func txCreateDeployment(groups []*dtypes.GroupSpec, id dtypes.DeploymentID) (err error) {
	res, err := config.SendMsgs([]sdk.Msg{dtypes.NewMsgCreateDeployment(id, groups)})
	if err != nil || res.Code != 0 {
		return err
	}

	logger.Info("create-deployment tx sent successfully", "hash", res.TxHash, "code", res.Code, "dseq", id.DSeq)
	return nil
}
