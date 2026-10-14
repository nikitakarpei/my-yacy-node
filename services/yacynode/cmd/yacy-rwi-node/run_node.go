package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
)

const shutdownTimeout = 15 * time.Second

const evictionInterval = time.Minute

const (
	escrowExpiryHoldFor   = 5 * time.Minute
	escrowExpiryBatchSize = 256
	escrowExpiryInterval  = time.Minute
)

type Listeners struct {
	PeerProtocol net.Listener
	Ops          net.Listener
}

func RunNode(
	ctx context.Context,
	config nodeconfiguration.Settings,
	listeners Listeners,
	vault *vault.Vault,
	registry *prometheus.Registry,
) error {
	assembledNode, err := assembleNode(ctx, config, vault, registry)
	if err != nil {
		return fmt.Errorf("assemble node: %w", err)
	}
	servers := []servergroup.NamedServer{
		{
			Name:     "peer protocol",
			Listener: listeners.PeerProtocol,
			Server:   serverFor(assembledNode.peerHandler),
		},
		{
			Name:     "ops",
			Listener: listeners.Ops,
			Server: serverFor(opsmetrics.NewMux(
				promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
			)),
		},
	}
	for _, server := range servers {
		slog.InfoContext(ctx, "serving",
			slog.String("service", server.Name),
			slog.String("addr", server.Listener.Addr().String()),
		)
	}

	loops := []func(context.Context) error{
		neverFailingLoop(assembledNode.peerAnnouncer.Run),
		neverFailingLoop(func(ctx context.Context) {
			eviction.RunSweepLoop(
				ctx,
				assembledNode.evictionSweeper,
				assembledNode.evictionObserver,
				evictionInterval,
			)
		}),
		neverFailingLoop(func(ctx context.Context) {
			rwiescrow.RunExpiryLoop(
				ctx,
				assembledNode.postingEscrow,
				assembledNode.postingEscrowObserver,
				rwiescrow.ExpiryConfig{
					HoldFor:  escrowExpiryHoldFor,
					Interval: escrowExpiryInterval,
					Batch:    escrowExpiryBatchSize,
				},
			)
		}),
	}
	if assembledNode.distributionLoop != nil {
		loops = append(loops, neverFailingLoop(assembledNode.distributionLoop.Run))
	}
	if assembledNode.pageOfferIntake != nil {
		defer assembledNode.pageOfferIntake.Close()

		loops = append(loops, assembledNode.pageOfferIntake.Run)
	}

	return servergroup.Run(ctx, shutdownTimeout, servers, loops...)
}

const serverReadHeaderTimeout = 10 * time.Second

func serverFor(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}
}

func neverFailingLoop(loop func(context.Context)) func(context.Context) error {
	return func(ctx context.Context) error {
		loop(ctx)

		return nil
	}
}
