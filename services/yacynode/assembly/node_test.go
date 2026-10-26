package yacynodeassembly_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	yacynode "github.com/nikitakarpei/yacy-rwi-node/yacynode/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	nodeHashText = "0123456789AB"
	nodeNetwork  = "testnet"
	settleFor    = 5 * time.Second
)

func TestRunServiceStopsWhenItsContextIsCanceled(t *testing.T) {
	node := startNode(t, nodeConfigFor(t))

	node.stop()

	if err := node.wait(t); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("RunService: %v", err)
	}
}

func TestRunServiceServesPeersWhileThePageOffersAreUnavailable(t *testing.T) {
	config := nodeConfigFor(t)
	config.PageOfferIntake = nodeconfiguration.PageOfferIntakeConfig{
		PageOfferNATSURL:           "nats://127.0.0.1:1",
		PageOfferDurable:           nodeconfiguration.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: nodeconfiguration.DefaultPageOfferIntakeConcurrency,
	}
	node := startNode(t, config)

	status, body := node.answerTo(t, http.MethodGet, "http://"+node.peerAddr+"/", nil)
	if status != http.StatusOK {
		t.Errorf(
			"landing page = %d %q, want 200 while the page offers are unavailable",
			status,
			body,
		)
	}

	node.stop()
	if err := node.wait(t); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("RunService: %v", err)
	}
}

func TestServedPeerRequestsAreCountedByEndpointAndStatus(t *testing.T) {
	node := startNode(t, nodeConfigFor(t))
	defer node.stop()

	node.get(t, "/")
	node.get(t, "/no-such-page")

	for _, served := range []struct{ code, endpoint string }{
		{code: "200", endpoint: "/{$}"},
		{code: "404", endpoint: "unmatched"},
	} {
		if counted := node.requestsCounted(t, served.code, served.endpoint); counted != 1 {
			t.Errorf(
				"requests counted for code %s at %s = %v, want 1",
				served.code,
				served.endpoint,
				counted,
			)
		}
	}
}

func TestHelloAnswersWithTheNodesOwnSeed(t *testing.T) {
	node := startNode(t, nodeConfigFor(t))
	defer node.stop()

	resp := node.hello(t, yacyproto.HelloRequest{
		NetworkName: nodeNetwork,
		Iam:         callerSeed(t).Hash,
		Seed:        callerSeed(t),
		Count:       1,
	})

	own, found := resp.OwnSeed().Get()
	if !found {
		t.Fatal("hello answered with no seed of its own")
	}
	if want := nodeHash(t); own.Hash != want {
		t.Errorf("own seed hash = %v, want %v", own.Hash, want)
	}
	if _, classified := resp.YourType.Get(); !classified {
		t.Error("hello left the caller unclassified, so it never read its own network name")
	}
}

func callerSeed(t *testing.T) yacymodel.Seed {
	t.Helper()

	hash, err := yacymodel.ParseHash("BA9876543210")
	if err != nil {
		t.Fatalf("parse caller hash: %v", err)
	}
	name, err := yacymodel.ParsePeerName("caller")
	if err != nil {
		t.Fatalf("parse caller name: %v", err)
	}
	host, err := yacymodel.ParseHost("203.0.113.9")
	if err != nil {
		t.Fatalf("parse caller host: %v", err)
	}
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		t.Fatalf("parse caller port: %v", err)
	}

	return yacymodel.Seed{
		Hash:        hash,
		Name:        name,
		PeerType:    yacymodel.PeerSenior,
		PrimaryHost: yacymodel.Some(host),
		Port:        yacymodel.Some(port),
	}
}

func nodeHash(t *testing.T) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(nodeHashText)
	if err != nil {
		t.Fatalf("parse node hash: %v", err)
	}

	return hash
}

func TestRunServicePublishesWhatStorageHolds(t *testing.T) {
	node := startNode(t, nodeConfigFor(t))
	defer node.stop()

	node.get(t, "/")
	for _, gauge := range []string{
		"yacynode_info",
		"yacynode_vault_quota_bytes",
		"yacynode_vault_used_bytes",
		"yacynode_vault_bucket_records",
	} {
		if published := node.seriesOf(t, gauge); published == 0 {
			t.Errorf("registry does not carry %s", gauge)
		}
	}
}

func TestRunServiceFailsWhenStorageCannotBeOpened(t *testing.T) {
	config := nodeConfigFor(t)
	config.Storage.Path = filepath.Join(fileIn(t), "node")

	err := yacynode.RunService(
		t.Context(),
		config,
		yacynode.Listeners{PeerProtocol: listenerFor(t)},
		prometheus.NewRegistry(),
	)
	if err == nil {
		t.Fatal("RunService returned nil, want the storage failure")
	}
}

func TestListenersForListensOnThePeerAddress(t *testing.T) {
	config := nodeConfigFor(t)
	config.Serving.PeerAddr = "127.0.0.1:0"

	listeners, err := yacynode.ListenersFor(t.Context(), config)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.PeerProtocol.Close() })
	if listeners.PeerProtocol.Addr().String() == "127.0.0.1:0" {
		t.Errorf("peer protocol listener = %s, want a bound port", listeners.PeerProtocol.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	config := nodeConfigFor(t)
	config.Serving.PeerAddr = "not-an-address"

	if _, err := yacynode.ListenersFor(t.Context(), config); err == nil {
		t.Fatal("expected error for an unusable peer address")
	}
}

func TestListenersCloseReleasesThePeerAddress(t *testing.T) {
	config := nodeConfigFor(t)
	config.Serving.PeerAddr = "127.0.0.1:0"
	listeners, err := yacynode.ListenersFor(t.Context(), config)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}

	if err := listeners.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := servergroup.Listen(t.Context(), listeners.PeerProtocol.Addr().String())
	if err != nil {
		t.Fatalf("listen on the closed peer address: %v", err)
	}
	_ = reopened.Close()
}

func nodeConfigFor(t *testing.T) yacynode.ServiceConfig {
	t.Helper()

	config, err := yacynode.LoadServiceConfig(envFrom(map[string]string{
		nodeconfiguration.EnvInitialPeerHash: nodeHashText,
		nodeconfiguration.EnvPeerName:        "node",
		nodeconfiguration.EnvNetworkName:     nodeNetwork,
		nodeconfiguration.EnvEgressProxyURL:  "http://127.0.0.1:1",
		nodeconfiguration.EnvDataDir:         t.TempDir(),
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	return config
}

func fileIn(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	return path
}

func listenerFor(t *testing.T) net.Listener {
	t.Helper()

	listener, err := servergroup.Listen(t.Context(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

type runningNode struct {
	peerAddr string
	registry *prometheus.Registry
	stop     context.CancelFunc
	stopped  chan error
	finished chan struct{}
}

func startNode(t *testing.T, config yacynode.ServiceConfig) runningNode {
	t.Helper()

	listeners := yacynode.Listeners{PeerProtocol: listenerFor(t)}
	ctx, cancel := context.WithCancel(context.Background())
	node := runningNode{
		peerAddr: listeners.PeerProtocol.Addr().String(),
		registry: prometheus.NewRegistry(),
		stop:     cancel,
		stopped:  make(chan error, 1),
		finished: make(chan struct{}),
	}
	go func() {
		defer close(node.finished)
		node.stopped <- yacynode.RunService(ctx, config, listeners, node.registry)
	}()
	t.Cleanup(func() { node.stopAndAwait(t) })

	return node
}

func (n runningNode) stopAndAwait(t *testing.T) {
	t.Helper()

	n.stop()
	select {
	case <-n.finished:
	case <-time.After(settleFor):
		t.Error("RunService did not stop after it was cancelled")
	}
}

func (n runningNode) wait(t *testing.T) error {
	t.Helper()

	select {
	case err := <-n.stopped:
		return err
	case <-time.After(settleFor):
		t.Fatal("RunService neither stopped nor failed")

		return nil
	}
}

func (n runningNode) get(t *testing.T, path string) {
	t.Helper()

	n.answerTo(t, http.MethodGet, "http://"+n.peerAddr+path, nil)
}

func (n runningNode) hello(t *testing.T, req yacyproto.HelloRequest) yacyproto.HelloResponse {
	t.Helper()

	status, body := n.answerTo(
		t,
		http.MethodPost,
		"http://"+n.peerAddr+yacyproto.PathHello,
		strings.NewReader(req.Form().Encode()),
	)
	if status != http.StatusOK {
		t.Fatalf("hello status = %d, want 200, body = %q", status, body)
	}

	parsed, err := yacyproto.ParseHelloResponse(
		context.Background(),
		yacyproto.ParseResponseFields(body),
	)
	if err != nil {
		t.Fatalf("ParseHelloResponse: %v", err)
	}

	return parsed
}

func (n runningNode) requestsCounted(t *testing.T, code, endpoint string) float64 {
	t.Helper()

	families, err := n.registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "yacynode_http_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["code"] == code && labels["endpoint"] == endpoint {
				return metric.GetCounter().GetValue()
			}
		}
	}

	return 0
}

func (n runningNode) seriesOf(t *testing.T, name string) int {
	t.Helper()

	published, err := testutil.GatherAndCount(n.registry, name)
	if err != nil {
		t.Fatalf("gather %s: %v", name, err)
	}

	return published
}

func (n runningNode) answerTo(
	t *testing.T,
	method, address string,
	sent io.Reader,
) (int, string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), settleFor)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, address, sent)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if sent != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		n.failUnanswered(t, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return resp.StatusCode, string(body)
}

func (n runningNode) failUnanswered(t *testing.T, err error) {
	t.Helper()

	select {
	case stopErr := <-n.stopped:
		t.Fatalf("RunService stopped before it served: %v", stopErr)
	default:
		t.Fatalf("node never answered: %v", err)
	}
}

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
