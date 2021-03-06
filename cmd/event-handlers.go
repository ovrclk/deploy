package cmd

import (
	"context"
	"fmt"
	"path"

	"github.com/ovrclk/akash/provider/gateway"
	"github.com/ovrclk/akash/provider/manifest"
	"github.com/ovrclk/akash/pubsub"
	dtypes "github.com/ovrclk/akash/x/deployment/types"
	mtypes "github.com/ovrclk/akash/x/market/types"
	pmodule "github.com/ovrclk/akash/x/provider"
	"gopkg.in/fsnotify.v1"
)

// EventHandler is a type of function that handles events coming out of the event bus
type EventHandler func(pubsub.Event) error

// SendManifestHander sends manifests on the lease created event
func SendManifestHander(dd *DeploymentData) func(pubsub.Event) error {
	return func(ev pubsub.Event) (err error) {
		addr := config.GetAccAddress()
		log := logger.With("action", "send-manifest")
		switch event := ev.(type) {
		// Handle Lease creation events
		case mtypes.EventLeaseCreated:
			if addr.Equals(event.ID.Owner) {
				pclient := pmodule.AppModuleBasic{}.GetQueryClient(config.CLICtx(config.NewTMClient()))
				provider, err := pclient.Provider(event.ID.Provider)
				if err != nil {
					return err
				}

				log.Info("sending manifest to provider", "provider", event.ID.Provider, "uri", provider.HostURI, "dseq", event.ID.DSeq)
				if err = gateway.NewClient().SubmitManifest(
					context.Background(),
					provider.HostURI,
					&manifest.SubmitRequest{
						Deployment: event.ID.DeploymentID(),
						Manifest:   dd.Manifest,
					},
				); err != nil {
					return err
				}
			}
		}
		return
	}
}

// DeploymentDataUpdateHandler updates a DeploymentData and prints relevant events
func DeploymentDataUpdateHandler(dd *DeploymentData) func(pubsub.Event) error {
	return func(ev pubsub.Event) (err error) {
		addr := dd.DeploymentID.Owner
		log := logger.With("addr", addr, "dseq", dd.DeploymentID.DSeq)
		switch event := ev.(type) {
		// Handle deployment creation events
		case dtypes.EventDeploymentCreated:
			if event.ID.Equals(dd.DeploymentID) {
				log.Info("deployment created")
			}
			return

		// Handle deployment update events
		case dtypes.EventDeploymentUpdated:
			if event.ID.Equals(dd.DeploymentID) {
				log.Info("deployment updated")
			}
			return

		// Handle deployment close events
		case dtypes.EventDeploymentClosed:
			if event.ID.Equals(dd.DeploymentID) {
				// TODO: Maybe we should exit here as the tracked deployment is now closed?
				log.Info("deployment closed")
			}
			return

		// Handle deployment group close events
		case dtypes.EventGroupClosed:
			if event.ID.Owner.Equals(addr) && event.ID.DSeq == dd.DeploymentID.DSeq {
				// TODO: Maybe more housekeeping here?
				log.Info("deployment group closed")
			}
			return

		// Handle Order creation events
		case mtypes.EventOrderCreated:
			if addr.Equals(event.ID.Owner) && event.ID.DSeq == dd.DeploymentID.DSeq {
				dd.AddOrder(event.ID)
				log.Info("order for deployment created", "oseq", event.ID.OSeq)
			}
			return

		// Handle Order close events
		case mtypes.EventOrderClosed:
			if addr.Equals(event.ID.Owner) && event.ID.DSeq == dd.DeploymentID.DSeq {
				dd.RemoveOrder(event.ID)
				log.Info("order for deployment closed", "oseq", event.ID.OSeq)
			}
			return

		// Handle Bid creation events
		case mtypes.EventBidCreated:
			if addr.Equals(event.ID.Owner) && event.ID.DSeq == dd.DeploymentID.DSeq {
				log.Info("bid for order created", "oseq", event.ID.OSeq, "price", event.Price)
			}
			return

		// Handle Bid close events
		case mtypes.EventBidClosed:
			if addr.Equals(event.ID.Owner) && event.ID.DSeq == dd.DeploymentID.DSeq {
				log.Info("bid for order closed", "oseq", event.ID.OSeq, "price", event.Price)
			}
			return

		// Handle Lease creation events
		case mtypes.EventLeaseCreated:
			if addr.Equals(event.ID.Owner) && event.ID.DSeq == dd.DeploymentID.DSeq {
				dd.AddLease(event.ID)
				log.Info("lease for order created", "oseq", event.ID.OSeq, "price", event.Price)
			}
			return

		// Handle Lease close events
		case mtypes.EventLeaseClosed:
			if addr.Equals(event.ID.Owner) && event.ID.DSeq == dd.DeploymentID.DSeq {
				dd.RemoveLease(event.ID)
				log.Info("lease for order closed", "oseq", event.ID.OSeq, "price", event.Price)
			}
			return

			// In any other case we should exit with error
		default:
			return fmt.Errorf("should be unreachable code, exit with this error")
		}
	}
}

// PrintHandler prints all the events
func PrintHandler(ev pubsub.Event) (err error) {
	addr := config.GetAccAddress()
	log := logger.With("addr", addr)
	switch event := ev.(type) {
	// Handle deployment creation events
	case dtypes.EventDeploymentCreated:
		if addr.Equals(event.ID.Owner) {
			log.Info("deployment created", "dseq", event.ID.DSeq)
		}
		return

	// Handle deployment update events
	case dtypes.EventDeploymentUpdated:
		if addr.Equals(event.ID.Owner) {
			log.Info("deployment updated", "dseq", event.ID.DSeq)
		}
		return

	// Handle deployment close events
	case dtypes.EventDeploymentClosed:
		if addr.Equals(event.ID.Owner) {
			log.Info("deployment closed", "dseq", event.ID.DSeq)
		}
		return

	// Handle deployment group close events
	case dtypes.EventGroupClosed:
		if addr.Equals(event.ID.Owner) {
			log.Info("deployment group closed", "dseq", event.ID.DSeq)
		}
		return

	// Handle Order creation events
	case mtypes.EventOrderCreated:
		if addr.Equals(event.ID.Owner) {
			log.Info("order for deployment created", "dseq", event.ID.DSeq, "oseq", event.ID.OSeq)
		}
		return

	// Handle Order close events
	case mtypes.EventOrderClosed:
		if addr.Equals(event.ID.Owner) {
			log.Info("order for deployment closed", "dseq", event.ID.DSeq, "oseq", event.ID.OSeq)
		}
		return

	// Handle Bid creation events
	case mtypes.EventBidCreated:
		if addr.Equals(event.ID.Owner) {
			log.Info("bid for order created", "dseq", event.ID.DSeq, "oseq", event.ID.OSeq, "price", event.Price)
		}
		return

	// Handle Bid close events
	case mtypes.EventBidClosed:
		if addr.Equals(event.ID.Owner) {
			log.Info("bid for order closed", "dseq", event.ID.DSeq, "oseq", event.ID.OSeq, "price", event.Price)
		}
		return

	// Handle Lease creation events
	case mtypes.EventLeaseCreated:
		if addr.Equals(event.ID.Owner) {
			log.Info("lease for order created", "dseq", event.ID.DSeq, "oseq", event.ID.OSeq, "price", event.Price)
		}
		return

	// Handle Lease close events
	case mtypes.EventLeaseClosed:
		if addr.Equals(event.ID.Owner) {
			log.Info("lease for order closed", "dseq", event.ID.DSeq, "oseq", event.ID.OSeq, "price", event.Price)
		}
		return

	// Handle filesystem events in the configuration directory
	// TODO: Handle "$CFG/deployemnts/*.yaml" events seperately from $CFG events
	case fsnotify.Event:
		return printFSEvents(event)

	// Handle filesystem errors by exiting
	case error:
		return event

	// In any other case we should exit with error
	default:
		return fmt.Errorf("should be unreachable code, exit with this error")
	}
}

// printFSEvents prints all filesystem events in the deployment directory
func printFSEvents(event fsnotify.Event) error {
	log := logger.With("events", "filesystem")
	switch {
	case path.Dir(event.Name) == path.Join(homePath, "deployments") && path.Ext(event.Name) == ".yaml":
		// TODO: New file created? we want to create a new deployment
		// TODO: File modified? we want to update an existing deployment
		// TODO: File moved? error and exit?
		// TODO: File deleted? close deployement, error and exit?
		switch event.Op {
		case fsnotify.Create:
			log.Info("deployment file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Write:
			log.Info("deployment file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Remove:
			log.Info("deployment file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Rename:
			log.Info("deployment file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Chmod:
			log.Info("deployment file", "file", path.Base(event.Name), "event", event.Op)
		}
		return nil
	case path.Dir(event.Name) == defaultHome:
		// TODO: Config file changed? warn changes not incorporated, error and exit?
		// TODO: Priv key file moved or changed? error and exit?
		switch event.Op {
		case fsnotify.Create:
			log.Info("config dir file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Write:
			log.Info("config dir file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Remove:
			log.Info("config dir file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Rename:
			log.Info("config dir file", "file", path.Base(event.Name), "event", event.Op)
		case fsnotify.Chmod:
			log.Info("config dir file", "file", path.Base(event.Name), "event", event.Op)
		}
		return nil
	default:
		log.Info("unexpected event", "file", path.Base(event.Name), "event", event.Op)
		return nil
	}
}

// // funcEventHandlerBoilerplate prints all the events
// func funcEventHandlerBoilerplate(ev pubsub.Event, client *rpchttp.HTTP) (err error) {
// 	log := logger.With("foo", bar)
// 	switch event := ev.(type) {
// 	case dtypes.EventDeploymentCreated:
// 	case dtypes.EventDeploymentUpdated:
// 	case dtypes.EventDeploymentClosed:
// 	case dtypes.EventGroupClosed:
// 	case mtypes.EventOrderCreated:
// 	case mtypes.EventOrderClosed:
// 	case mtypes.EventBidCreated:
// 	case mtypes.EventBidClosed:
// 	case mtypes.EventLeaseCreated:
// 	case mtypes.EventLeaseClosed:
// 	case fsnotify.Event:
// 	case error:
// 	default:
// 	}
// }
