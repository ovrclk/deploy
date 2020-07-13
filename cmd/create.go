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
	dcli "github.com/ovrclk/akash/x/deployment/client/cli"
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
			log := logger.With("cli", "create")
			dd, err := NewDeploymentData(args[0], cmd.Flags(), config.GetAccAddress())
			if err != nil {
				return err
			}

			ctx, _ := context.WithCancel(context.Background())
			group, _ := errgroup.WithContext(ctx)

			// Listen to on chain events and send the manifest when required
			group.Go(func() error {
				if err = ChainEmitter(ctx, DeploymentDataUpdateHandler(dd), SendManifestHander(dd, config.NewTMClient())); err != nil {
					log.Error("error watching events", err)
				}
				return err
			})

			// Store the deployment manifest in the archive
			group.Go(func() error {
				if err = createDeploymentFileInArchive(dd); err != nil {
					log.Error("error updating archive", err)
				}
				return err
			})

			// Send the deployment creation transaction
			group.Go(func() error {
				if err = txCreateDeployment(dd); err != nil {
					log.Error("error creating deployment", err)
				}
				return err
			})

			// Wait for the leases to be created and then start polling the provider for service availability
			group.Go(func() error {
				if err = waitForLeasesAndPollService(); err != nil {
					log.Error("error listening for service", err)
				}
				return err
			})

			return group.Wait()
		},
	}
	dcli.AddDeploymentIDFlags(cmd.Flags())
	return cmd
}

func waitForLeasesAndPollService() error {
	return nil
}

func createDeploymentFileInArchive(dd *DeploymentData) error {
	fileName := fmt.Sprintf("%s.%d.yaml", dd.DeploymentID.Owner, dd.DeploymentID.DSeq)
	return ioutil.WriteFile(path.Join(homePath, "deployments", fileName), dd.SDLFile, 666)
}

func txCreateDeployment(dd *DeploymentData) (err error) {
	res, err := config.SendMsgs([]sdk.Msg{dd.MsgCreate()})
	if err != nil || res.Code != 0 {
		logger.Error("create-deployment tx failed", "hash", res.TxHash, "code", res.Code, "dseq", dd.DeploymentID.DSeq)
		return err
	}

	logger.Info("create-deployment tx sent successfully", "hash", res.TxHash, "code", res.Code, "dseq", dd.DeploymentID.DSeq)
	return nil
}
