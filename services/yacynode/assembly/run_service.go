package yacynodeassembly

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault/pebblemetrics"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peertls"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
)

const metricNamespace = "yacynode"

const shutdownTimeout = 15 * time.Second

const evictionInterval = time.Minute

const (
	escrowExpiryHoldFor   = 5 * time.Minute
	escrowExpiryBatchSize = 256
	escrowExpiryInterval  = time.Minute
)

type Listeners struct {
	PeerProtocol       net.Listener
	SecurePeerProtocol yacymodel.Optional[net.Listener]
}

func ListenersFor(ctx context.Context, cfg ServiceConfig) (Listeners, error) {
	peerProtocol, err := servergroup.Listen(ctx, cfg.Serving.PeerAddr)
	if err != nil {
		return Listeners{}, err
	}
	securePeerAddr, serving := cfg.Serving.SecurePeerAddr.Get()
	if !serving {
		return Listeners{PeerProtocol: peerProtocol}, nil
	}
	securePeerProtocol, err := servergroup.Listen(ctx, securePeerAddr)
	if err != nil {
		return Listeners{}, errors.Join(err, peerProtocol.Close())
	}
	return Listeners{
		PeerProtocol:       peerProtocol,
		SecurePeerProtocol: yacymodel.Some(securePeerProtocol),
	}, nil
}

func (l Listeners) Close() error {
	var errs []error
	if err := l.PeerProtocol.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close the peer protocol listener: %w", err))
	}
	if securePeerProtocol, serving := l.SecurePeerProtocol.Get(); serving {
		if err := securePeerProtocol.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close the secure peer protocol listener: %w", err))
		}
	}
	return errors.Join(errs...)
}

func RunService(
	ctx context.Context,
	cfg ServiceConfig,
	listeners Listeners,
	registry *prometheus.Registry,
) error {
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "yacynode_info",
		Help: "YaCy node application identity.",
	}, func() float64 { return 1 }))
	storage, err := storageFor(cfg.Storage, registry)
	if err != nil {
		return err
	}
	defer closeVault(storage)

	return runNode(ctx, cfg, listeners, storage, registry)
}

func storageFor(
	config nodeconfiguration.StorageConfig,
	registry *prometheus.Registry,
) (*vault.Vault, error) {
	engine, err := pebblevault.OpenEngine(
		config.Path,
		config.QuotaByte,
		pebblevault.MachineLimits{
			BlockCacheBytes:       config.BlockCacheByte,
			MemtableBytes:         config.MemtableByte,
			CompactionConcurrency: config.CompactionConcurrency,
			OpenFileLimit:         config.OpenFileLimit,
			WriteConcurrency:      config.WriteConcurrency,
		},
		pebblemetrics.NewWriteStallMetrics(registry, metricNamespace),
		pebblemetrics.NewWriteConflictMetrics(registry, metricNamespace),
	)
	if err != nil {
		return nil, fmt.Errorf("open storage: %w", err)
	}
	pebblemetrics.NewConditionMetrics(registry, metricNamespace, engine)

	storage, err := vault.New(
		engine,
		vaultmetrics.NewTransactionMetrics(registry, metricNamespace),
		nil,
		wallclock.Clock{},
	)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("lend storage: %w", err), engine.Close())
	}
	return storage, nil
}

func closeVault(storage *vault.Vault) {
	if err := storage.Close(); err != nil {
		slog.ErrorContext(context.Background(), "storage close failed", slog.Any("error", err))
	}
}

func runNode(
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
	servers, err := peerProtocolServersFor(config.Identity, listeners, assembledNode.peerHandler)
	if err != nil {
		return err
	}
	for _, server := range servers {
		slog.InfoContext(ctx, "serving",
			slog.String("server", server.Name),
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

func peerProtocolServersFor(
	identity nodeconfiguration.IdentityConfig,
	listeners Listeners,
	peerHandler http.Handler,
) ([]servergroup.NamedServer, error) {
	peerProtocolServer := servergroup.NamedServer{
		Name:     "peer protocol",
		Listener: listeners.PeerProtocol,
		Server:   serverFor(peerHandler),
	}
	securePeerProtocol, serving := listeners.SecurePeerProtocol.Get()
	if !serving {
		return []servergroup.NamedServer{peerProtocolServer}, nil
	}
	tlsConfig, err := peertls.ServerConfigFor(identity.PeerKey, identity.NetworkName)
	if err != nil {
		return nil, fmt.Errorf("secure peer protocol: %w", err)
	}

	return []servergroup.NamedServer{peerProtocolServer, {
		Name:     "secure peer protocol",
		Listener: tls.NewListener(securePeerProtocol, tlsConfig),
		Server:   serverFor(peerHandler),
	}}, nil
}

const (
	serverReadHeaderTimeout = 10 * time.Second
	serverReadTimeout       = 30 * time.Second
	serverIdleTimeout       = time.Minute
)

func serverFor(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		IdleTimeout:       serverIdleTimeout,
	}
}

func neverFailingLoop(loop func(context.Context)) func(context.Context) error {
	return func(ctx context.Context) error {
		loop(ctx)

		return nil
	}
}
