package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
)

func main() {
	if err := run(); err != nil {
		slog.ErrorContext(context.Background(), "node terminated", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	if err := applog.Configure(os.Getenv); err != nil {
		return fmt.Errorf("configure logging: %w", err)
	}

	config, err := nodeconfiguration.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("load node config: %w", err)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Name: "yacynode_info",
			Help: "YaCy node application identity.",
		}, func() float64 { return 1 }),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	storage, err := pebblevault.Open(
		config.Storage.Path,
		config.Storage.QuotaByte,
		pebblevault.MachineLimits{
			BlockCacheBytes:       config.Storage.BlockCacheByte,
			MemtableBytes:         config.Storage.MemtableByte,
			CompactionConcurrency: config.Storage.CompactionConcurrency,
			OpenFileLimit:         config.Storage.OpenFileLimit,
		},
		metrics.NewVaultTransactionMetrics(registry),
	)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer closeVault(storage)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return RunNode(ctx, config, storage, registry)
}

func closeVault(storage *vault.Vault) {
	if err := storage.Close(); err != nil {
		slog.ErrorContext(context.Background(), "storage close failed", slog.Any("error", err))
	}
}
