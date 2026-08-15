package main

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/metrics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultengines/memory"
)

func TestRunRejectsInvalidConfig(t *testing.T) {
	t.Setenv(EnvPeerHash, "")
	if err := run(); err == nil {
		t.Fatal("expected error for missing config")
	}
}

func testConfig(t *testing.T) NodeConfig {
	t.Helper()

	config, err := LoadNodeConfig(func(key string) string {
		switch key {
		case EnvPeerHash:
			return "0123456789AB"
		case EnvPeerName:
			return "node"
		case EnvAdvertiseHost:
			return "203.0.113.1"
		case EnvDataDir:
			return t.TempDir()
		case EnvProxyURL:
			return "http://proxy:4750"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	return config
}

func openTestVault(t *testing.T) *vault.Vault {
	t.Helper()

	v, err := memory.Open(0, nil)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	return v
}

func assembleTestNode(t *testing.T, config NodeConfig, vault *vault.Vault) node {
	t.Helper()

	assembled, err := assembleNode(
		context.Background(),
		config,
		vault,
		newEgressProxyClient(config.ProxyURL, outboundRequestTimeout),
		metrics.NewDistributionMetrics(prometheus.NewRegistry()),
		metrics.NewDHTRingMetrics(prometheus.NewRegistry()),
		metrics.NewPeerRosterMetrics(prometheus.NewRegistry()),
		metrics.NewRWIEscrowMetrics(prometheus.NewRegistry()),
		searchmetrics.NewSearchMetrics(prometheus.NewRegistry()),
	)
	if err != nil {
		t.Fatalf("assemble: %v", err)
	}

	return assembled
}

func TestServeReturnsNilAfterCancel(t *testing.T) {
	assembled := assembleTestNode(t, testConfig(t), openTestVault(t))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := serve(
		ctx,
		assembled,
		metrics.NewEvictionMetrics(prometheus.NewRegistry()),
		metrics.NewRWIEscrowMetrics(prometheus.NewRegistry()),
		servergroup.NamedServer{
			Name:   "peer protocol",
			Server: buildServer("127.0.0.1:0", assembled.peerMux),
		},
		servergroup.NamedServer{
			Name: "ops",
			Server: buildServer(
				"127.0.0.1:0",
				opsmetrics.NewMux(
					promhttp.HandlerFor(prometheus.NewRegistry(), promhttp.HandlerOpts{}),
				),
			),
		},
	)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
}

func TestServeShutsDownOnListenError(t *testing.T) {
	assembled := assembleTestNode(t, testConfig(t), openTestVault(t))

	err := serve(
		context.Background(),
		assembled,
		metrics.NewEvictionMetrics(prometheus.NewRegistry()),
		metrics.NewRWIEscrowMetrics(prometheus.NewRegistry()),
		servergroup.NamedServer{
			Name:   "peer protocol",
			Server: buildServer("203.0.113.255:-1", assembled.peerMux),
		},
	)
	if err == nil {
		t.Fatal("expected listen error")
	}
}
