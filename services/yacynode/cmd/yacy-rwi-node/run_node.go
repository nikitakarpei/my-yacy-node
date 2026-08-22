package main

import (
	"context"
	"fmt"
	"log/slog"
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

func RunNode(
	ctx context.Context,
	config nodeconfiguration.Settings,
	vault *vault.Vault,
	registry *prometheus.Registry,
) error {
	assembledNode, err := assembleNode(ctx, config, vault, registry)
	if err != nil {
		return fmt.Errorf("assemble node: %w", err)
	}
	servers := []servergroup.NamedServer{
		{
			Name:   "peer protocol",
			Server: serverOn(config.Serving.PeerAddr, assembledNode.peerHandler),
		},
		{Name: "ops", Server: serverOn(config.Serving.OpsAddr, opsmetrics.NewMux(
			promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		))},
	}
	for _, server := range servers {
		slog.InfoContext(ctx, "serving",
			slog.String("service", server.Name),
			slog.String("addr", server.Server.Addr),
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
	if assembledNode.distributionCycle != nil {
		loops = append(loops, neverFailingLoop(assembledNode.distributionCycle.Run))
	}
	if assembledNode.scrapeRequestIngest != nil {
		defer assembledNode.scrapeRequestIngest.Close()

		loops = append(loops, assembledNode.scrapeRequestIngest.Run)
	}

	return servergroup.Run(ctx, shutdownTimeout, servers, loops...)
}

const serverReadHeaderTimeout = 10 * time.Second

func serverOn(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
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
