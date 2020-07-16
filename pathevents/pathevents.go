package pathevents

import (
	"context"
	"fmt"
	"os"

	"github.com/ovrclk/akash/pubsub"
	"github.com/tendermint/tendermint/libs/log"
	"golang.org/x/sync/errgroup"
	"gopkg.in/fsnotify.v1"
)

// Publish publishes filesystem events for pth and path.Join(pth, 'deployments') to the passed bus
func Publish(ctx context.Context, watcher *fsnotify.Watcher, pths []string, bus pubsub.Bus) error {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("events", "filesystem")
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
	loop:
		for {
			select {
			case <-ctx.Done():
				break loop
			case event, ok := <-watcher.Events:
				if !ok {
					err = fmt.Errorf("event not OK")
					break loop
				}
				bus.Publish(event)
			case err, ok := <-watcher.Errors:
				bus.Publish(err)
				if !ok || err != nil {
					break loop
				}
			}
		}
		return err
	})

	var err error
	for _, pth := range pths {
		if err = watcher.Add(pth); err != nil {
			logger.Error("failed to watch path", "error", err.Error(), "path", pth)
			return err
		}
		logger.Info("watching files", "path", pth)
	}

	return group.Wait()
}
