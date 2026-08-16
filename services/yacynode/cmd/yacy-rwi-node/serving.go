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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
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
	metrics.NewVaultCapacityMetrics(registry, vault)
	metrics.NewVaultCollectionMetrics(registry, vault)

	assembled, err := assembleNode(ctx, config, vault, registry)
	if err != nil {
		return fmt.Errorf("assemble node: %w", err)
	}
	metrics.NewRWIEscrowCapacityMetrics(registry, assembled.escrow)
	if assembled.crawl != nil {
		defer assembled.crawl.Close()
	}

	servers := []servergroup.NamedServer{
		{Name: "peer protocol", Server: serverOn(config.Serving.PeerAddr, assembled.peerHandler)},
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
		neverFailingLoop(assembled.announcer.Run),
		neverFailingLoop(func(ctx context.Context) {
			eviction.RunSweepLoop(
				ctx,
				assembled.sweeper,
				assembled.evictionObserver,
				evictionInterval,
			)
		}),
		neverFailingLoop(func(ctx context.Context) {
			rwiescrow.RunExpiryLoop(
				ctx,
				assembled.escrow,
				assembled.escrowObserver,
				rwiescrow.ExpiryConfig{
					HoldFor:  escrowExpiryHoldFor,
					Interval: escrowExpiryInterval,
					Batch:    escrowExpiryBatchSize,
				},
			)
		}),
	}
	if assembled.distributionCycle != nil {
		loops = append(loops, neverFailingLoop(assembled.distributionCycle.Run))
	}
	if assembled.crawl != nil {
		loops = append(loops, neverFailingLoop(assembled.crawl.Run))
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
