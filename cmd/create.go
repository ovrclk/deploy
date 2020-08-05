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
	"os"
	"path"
	"strings"
	"time"

	"github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ovrclk/akash/provider/cluster"
	"github.com/ovrclk/akash/provider/gateway"
	dcli "github.com/ovrclk/akash/x/deployment/client/cli"
	pmodule "github.com/ovrclk/akash/x/provider"
	pquery "github.com/ovrclk/akash/x/provider/query"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var (
	flagGasAdj    = "gas-adjustment"
	flagGasPrices = "gas-prices"
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
			if err := config.SetGasOnConfigFromFlags(cmd); err != nil {
				return err
			}

			log := logger.With("cli", "create")
			dd, err := NewDeploymentData(args[0], cmd.Flags(), config.GetAccAddress())
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			group, _ := errgroup.WithContext(ctx)

			// Listen to on chain events and send the manifest when required
			group.Go(func() error {
				if err = ChainEmitter(ctx, DeploymentDataUpdateHandler(dd), SendManifestHander(dd)); err != nil {
					log.Error("error watching events", err)
				}
				return err
			})

			// Store the deployment manifest in the archive
			group.Go(func() error {
				if err = config.CreateDeploymentFileInArchive(dd); err != nil {
					log.Error("error updating archive", err)
				}
				return err
			})

			// Send the deployment creation transaction
			group.Go(func() error {
				if err = config.TxCreateDeployment(dd); err != nil {
					log.Error("error creating deployment", err)
				}
				return err
			})

			// Wait for the leases to be created and then start polling the provider for service availability
			group.Go(func() error {
				if err = config.WaitForLeasesAndPollService(dd, cancel); err != nil {
					log.Error("error listening for service", err)
				}
				return err
			})

			return group.Wait()
		},
	}
	dcli.AddDeploymentIDFlags(cmd.Flags())
	rootCmd.PersistentFlags().Float64P(flagGasAdj, "a", 1.0, "gas adjustment for transactions. if your transactions are failing due to out of gas errors increase this number")
	rootCmd.PersistentFlags().StringP(flagGasPrices, "p", "0.025akash", "price for gas")
	if err := viper.BindPFlag(flagGasAdj, cmd.Flags().Lookup(flagGasAdj)); err != nil {
		panic(err)
	}
	if err := viper.BindPFlag(flagGasPrices, cmd.Flags().Lookup(flagGasPrices)); err != nil {
		panic(err)
	}
	return cmd
}

// SetGasOnConfigFromFlags pulls the gas prices and gas adj variables from the flags and sets them on the global config object
func (c *Config) SetGasOnConfigFromFlags(cmd *cobra.Command) error {
	gasAdj, err := cmd.Flags().GetFloat64(flagGasAdj)
	if err != nil {
		return err
	}
	gp, err := cmd.Flags().GetString(flagGasPrices)
	if err != nil {
		return err
	}
	gasPr, err := sdk.ParseDecCoins(gp)
	if err != nil {
		return err
	}
	c.gasPrices = gasPr
	c.gasAdj = gasAdj
	return nil
}

// WaitForLeasesAndPollService waits for
func (c *Config) WaitForLeasesAndPollService(dd *DeploymentData, cancel context.CancelFunc) error {
	log := logger
	pclient := pmodule.AppModuleBasic{}.GetQueryClient(config.CLICtx(config.NewTMClient()))
	timeout := time.After(90 * time.Second)
	tick := time.Tick(500 * time.Millisecond)
	for {
		select {
		case <-timeout:
			log.Info("timed out (90s) listening for deployment to be available")
			cancel()
			return nil
		case <-tick:
			if dd.ExpectedLeases() {
				for _, l := range dd.Leases() {

					var (
						p   *pquery.Provider
						err error
					)
					if err := retry.Do(func() error {
						p, err = pclient.Provider(l.Provider)
						if err != nil {
							// TODO: Log retry?
							return err
						}

						return nil
					}); err != nil {
						cancel()
						return fmt.Errorf("error querying provider: %w", err)
					}

					// TODO: Move to using service status here?
					var ls *cluster.LeaseStatus
					if err := retry.Do(func() error {
						ls, err = gateway.NewClient().LeaseStatus(context.Background(), p.HostURI, l)
						if err != nil {
							return err
						}
						return nil
					}); err != nil {
						cancel()
						return fmt.Errorf("error querying lease status: %w", err)
					}

					for _, s := range ls.Services {
						// TODO: Much better logging/ux could be put in here: waiting, timeouts etc...
						if s.Available == s.Total {
							log.Info(strings.Join(s.URIs, ","), "name", s.Name, "available", s.Available)
							cancel()
							return nil
						}
					}
				}
			}
		}
	}
}

// CreateDeploymentFileInArchive creates the deployment file in the `$HOME/.akash-deploy/deployments/` folder
func (c *Config) CreateDeploymentFileInArchive(dd *DeploymentData) error {
	fileName := fmt.Sprintf("%s.%d.yaml", dd.DeploymentID.Owner, dd.DeploymentID.DSeq)
	depDir := path.Join(homePath, "deployments")
	if _, err := os.Stat(depDir); os.IsNotExist(err) {
		if err = os.MkdirAll(depDir, 0777); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(path.Join(depDir, fileName), dd.SDLFile, 644)
}

// TxCreateDeployment takes DeploymentData and creates the specified deployment
func (c *Config) TxCreateDeployment(dd *DeploymentData) (err error) {
	res, err := c.SendMsgs([]sdk.Msg{dd.MsgCreate()})
	log := logger.With(
		"hash", res.TxHash,
		"code", res.Code,
		"codespace", res.Codespace,
		"action", "create-deployment",
		"dseq", dd.DeploymentID.DSeq,
	)

	if err != nil || res.Code != 0 {
		log.Error("tx failed")
		return err
	}

	log.Info("tx sent successfully")
	return nil
}
