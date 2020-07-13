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

	"github.com/jackzampolin/deploy/pathevents"
	"github.com/ovrclk/akash/cmd/common"
	"github.com/ovrclk/akash/events"
	"github.com/ovrclk/akash/pubsub"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gopkg.in/fsnotify.v1"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

// startCmd represents the watch command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Listen to the chain and configuration directory and print those events",
	RunE: func(cmd *cobra.Command, args []string) error {
		return common.RunForever(PrintChainAndFSEvents)
	},
}

// PrintChainAndFSEvents prints for all events created by this stream
func PrintChainAndFSEvents(ctx context.Context) error {
	return WatchForChainAndFSEvents(ctx, PrintBusEvents)
}

// WatchForChainAndFSEvents watches for a set of chain and filesystem
// events and takes actions based on them.
func WatchForChainAndFSEvents(ctx context.Context, ehs ...EventHandler) error {
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
				for _, eh := range ehs {
					if err = eh(ev, client); err != nil {
						return err
					}
				}
			}
		}
	})

	return group.Wait()
}
