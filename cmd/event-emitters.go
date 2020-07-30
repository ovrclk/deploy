package cmd

import (
	"context"
	"path"

	"github.com/ovrclk/akash/events"
	"github.com/ovrclk/akash/pubsub"
	"github.com/ovrclk/deploy/pathevents"
	"golang.org/x/sync/errgroup"
	"gopkg.in/fsnotify.v1"
)

// EventEmitter is a type that describes event emitter functions
type EventEmitter func(context.Context, ...EventHandler) error

// ChainAndFSEmitter runs the passed EventHandlers the on chain and filesystem event streams
func ChainAndFSEmitter(paths []string) func(context.Context, ...EventHandler) error {
	return func(ctx context.Context, ehs ...EventHandler) error {
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
						if err = eh(ev); err != nil {
							return err
						}
					}
				}
			}
		})

		return group.Wait()
	}
}

// ChainEmitter runs the passed EventHandlers just on the on chain event stream
func ChainEmitter(ctx context.Context, ehs ...EventHandler) (err error) {
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
					if err = eh(ev); err != nil {
						return err
					}
				}
			}
		}
	})

	return group.Wait()
}

// FSEvents runs the passed EventHandlers just on the filesystem event stream
func FSEvents(paths []string) func(context.Context, ...EventHandler) error {
	return func(ctx context.Context, ehs ...EventHandler) error {
		// Start the filesystem watcher
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return err
		}
		defer watcher.Close()

		// Start the pubsub bus
		bus := pubsub.NewBus()
		defer bus.Close()

		// Initialize a new error group
		group, ctx := errgroup.WithContext(ctx)

		// Publish filesystem events to the bus
		group.Go(func() error {
			return pathevents.Publish(ctx, watcher, paths, bus)
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
						if err = eh(ev); err != nil {
							return err
						}
					}
				}
			}
		})

		return group.Wait()
	}
}
