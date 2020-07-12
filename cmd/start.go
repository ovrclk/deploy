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
	"log"
	"path"

	"github.com/jackzampolin/deploy/pathevents"
	"github.com/ovrclk/akash/cmd/common"
	"github.com/ovrclk/akash/events"
	"github.com/ovrclk/akash/provider/gateway"
	"github.com/ovrclk/akash/provider/manifest"
	"github.com/ovrclk/akash/pubsub"
	"github.com/ovrclk/akash/sdl"
	dtypes "github.com/ovrclk/akash/x/deployment/types"
	mtypes "github.com/ovrclk/akash/x/market/types"
	pmodule "github.com/ovrclk/akash/x/provider"
	"github.com/spf13/cobra"
	rpchttp "github.com/tendermint/tendermint/rpc/client/http"
	"golang.org/x/sync/errgroup"
	"gopkg.in/fsnotify.v1"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

// startCmd represents the watch command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Listen to the chain and configuration directory and take actions based on changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return common.RunForever(WatchForChainAndFSEvents)
	},
}

// WatchForChainAndFSEvents watches for a set of chain and filesystem
// events and takes actions based on them.
func WatchForChainAndFSEvents(ctx context.Context) error {
	// Start the filesystem watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Instantiate and start tendermint RPC client
	client := config.NewTMClient()
	if err = client.Start(); err != nil {
		return err
	}

	// Start the pubsub bus
	bus := pubsub.NewBus()
	defer bus.Close()

	// Initialize a new error group
	group, ctx := errgroup.WithContext(ctx)

	// Publish chain events to the pubsub bus
	group.Go(func() error {
		return events.Publish(ctx, client, "akash-deploy", bus)
	})

	// Publish filesystem events to the bus
	group.Go(func() error {
		return pathevents.Publish(ctx, watcher, []string{
			homePath,
			path.Join(homePath, "deployments"),
		}, bus)
	})

	// Subscribe to the bus events
	subscriber, err := bus.Subscribe()
	if err != nil {
		return err
	}

	// Handle all the events coming out of the bus
	group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-subscriber.Done():
				return nil
			case ev := <-subscriber.Events():
				if err = handleBusEvents(ev, client); err != nil {
					return err
				}
			}
		}
	})

	return group.Wait()
}

func handleBusEvents(ev pubsub.Event, client *rpchttp.HTTP) (err error) {
	switch event := ev.(type) {
	// Handle deployment creation events
	case dtypes.EventDeploymentCreated:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Deployment %d created...", event.ID.DSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle deployment update events
	case dtypes.EventDeploymentUpdated:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Deployment %d updated...", event.ID.DSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle deployment close events
	case dtypes.EventDeploymentClosed:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Deployment %d closed...", event.ID.DSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle deployment group close events
	case dtypes.EventGroupClosed:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Deployment Group %d closed..\n", event.ID.DSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle Order creation events
	case mtypes.EventOrderCreated:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Order %d for deployemen%d created...\n", event.ID.OSeq, event.ID.DSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle Order close events
	case mtypes.EventOrderClosed:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Order %d for deployemen%d closed...\n", event.ID.OSeq, event.ID.DSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle Bid creation events
	case mtypes.EventBidCreated:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Bid of %s for order %d:%d created...\n", event.Price, event.ID.DSeq, event.ID.OSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle Bid close events
	case mtypes.EventBidClosed:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Bid of %s for order %d:%d closed...\n", event.Price, event.ID.DSeq, event.ID.OSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle Lease creation events
	case mtypes.EventLeaseCreated:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Lease for order %d:%d created...\n", event.ID.DSeq, event.ID.OSeq)
			pclient := pmodule.AppModuleBasic{}.GetQueryClient(config.CLICtx(client))
			provider, err := pclient.Provider(event.ID.Provider)
			if err != nil {
				return err
			}

			manifestFile := fmt.Sprintf("%s.%d.yaml", config.GetAccAddress().String(), event.ID.DSeq)
			dep, err := sdl.ReadFile(path.Join(homePath, "deployments", manifestFile))
			if err != nil {
				return err
			}

			mani, err := dep.Manifest()
			if err != nil {
				return err
			}

			fmt.Printf("Sending manifest to provider %s...\n", event.ID.Provider)
			if err = gateway.NewClient().SubmitManifest(
				context.Background(),
				provider.HostURI,
				&manifest.SubmitRequest{
					Deployment: event.ID.DeploymentID(),
					Manifest:   mani,
				},
			); err != nil {
				return err
			}

		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle Lease close events
	case mtypes.EventLeaseClosed:
		if config.GetAccAddress().Equals(event.ID.Owner) {
			log.Printf("Lease for order %d:%d closed..\n", event.ID.DSeq, event.ID.OSeq)
		} else {
			log.Printf("Event not for wallet: %d", event.ID.DSeq)
		}
		return

	// Handle filesystem events in the configuration directory
	// TODO: Handle "$CFG/deployemnts/*.yaml" events seperately from $CFG events
	case fsnotify.Event:
		return handleFSEvents(event)

	// Handle filesystem errors by exiting
	case error:
		return event

	// In any other case we should exit with error
	default:
		return fmt.Errorf("should be unreachable code, exit with this error")
	}
}

func handleFSEvents(event fsnotify.Event) error {
	switch {
	case path.Dir(event.Name) == path.Join(homePath, "deployments") && path.Ext(event.Name) == ".yaml":
		// TODO: New file created? we want to create a new deployment
		// TODO: File modified? we want to update an existing deployment
		// TODO: File moved? error and exit?
		// TODO: File deleted? close deployement, error and exit?
		switch event.Op {
		case fsnotify.Create:
			log.Printf("Deployment file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Write:
			log.Printf("Deployment file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Remove:
			log.Printf("Deployment file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Rename:
			log.Printf("Deployment file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Chmod:
			log.Printf("Deployment file %s: %s", path.Base(event.Name), event.Op)
		}
		return nil
	case path.Dir(event.Name) == defaultHome:
		// TODO: Config file changed? warn changes not incorporated, error and exit?
		// TODO: Priv key file moved or changed? error and exit?
		switch event.Op {
		case fsnotify.Create:
			log.Printf("Home dir file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Write:
			log.Printf("Home dir file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Remove:
			log.Printf("Home dir file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Rename:
			log.Printf("Home dir file %s: %s", path.Base(event.Name), event.Op)
		case fsnotify.Chmod:
			log.Printf("Home dir file %s: %s", path.Base(event.Name), event.Op)
		}
		return nil
	default:
		log.Printf("Unexpected event for file %s", event.Name)
		return nil
	}
}
