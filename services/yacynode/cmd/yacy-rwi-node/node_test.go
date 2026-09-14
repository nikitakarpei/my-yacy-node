package main_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	yacynode "github.com/nikitakarpei/yacy-rwi-node/yacynode/cmd/yacy-rwi-node"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/nodeconfiguration"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	nodeHashText = "0123456789AB"
	nodeNetwork  = "testnet"
	settleFor    = 5 * time.Second
)

func TestRunNodeStopsWhenItsContextIsCanceled(t *testing.T) {
	node := startNode(t, nodeConfigFor(t))

	node.stop()

	if err := node.wait(t); err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("RunNode: %v", err)
	}
}

func TestRunNodeReportsAnUnusableListenAddress(t *testing.T) {
	config := nodeConfigFor(t)
	config.Serving.PeerAddr = "203.0.113.255:-1"

	node := startNode(t, config)
	defer node.stop()

	if err := node.wait(t); err == nil {
		t.Fatal("RunNode returned nil, want the listen failure")
	}
}

func TestRunNodeReportsAnUnreachablePageOfferBroker(t *testing.T) {
	config := nodeConfigFor(t)
	config.PageOfferIntake = nodeconfiguration.PageOfferIntakeConfig{
		PageOfferNATSURL:           "nats://127.0.0.1:1",
		PageOfferDurable:           nodeconfiguration.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: nodeconfiguration.DefaultPageOfferIntakeConcurrency,
	}

	node := startNode(t, config)
	defer node.stop()

	err := node.wait(t)
	if err == nil {
		t.Fatal("RunNode returned nil, want the broker failure")
	}
	if !strings.Contains(err.Error(), "page offer broker") {
		t.Fatalf("RunNode: %v, want a page offer broker failure", err)
	}
}

func TestServedPeerRequestsAreCountedByEndpointAndStatus(t *testing.T) {
	node := startNode(t, nodeConfigFor(t))
	defer node.stop()

	node.get(t, "/")
	node.get(t, "/no-such-page")

	published := node.metrics(t)
	for _, counter := range []string{
		`yacynode_http_requests_total{code="200",endpoint="/{$}"} 1`,
		`yacynode_http_requests_total{code="404",endpoint="unmatched"} 1`,
	} {
		if !strings.Contains(published, counter) {
			t.Errorf("metrics do not carry %s", counter)
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
		Hash:           hash,
		Name:           name,
		PeerType:       yacymodel.PeerSenior,
		PrimaryAddress: yacymodel.Some(host),
		Port:           yacymodel.Some(port),
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

func TestOpsEndpointPublishesWhatStorageHolds(t *testing.T) {
	node := startNode(t, nodeConfigFor(t))
	defer node.stop()

	published := node.metrics(t)
	for _, gauge := range []string{"yacynode_vault_quota_bytes", "yacynode_vault_used_bytes", "yacynode_vault_collection"} {
		if !strings.Contains(published, gauge) {
			t.Errorf("metrics do not carry %s", gauge)
		}
	}
}

func nodeConfigFor(t *testing.T) nodeconfiguration.Settings {
	t.Helper()

	config, err := nodeconfiguration.Load(envFrom(map[string]string{
		nodeconfiguration.EnvInitialPeerHash: nodeHashText,
		nodeconfiguration.EnvPeerName:        "node",
		nodeconfiguration.EnvNetworkName:     nodeNetwork,
		nodeconfiguration.EnvEgressProxyURL:  "http://127.0.0.1:1",
	}))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	config.Serving.PeerAddr = reservedAddr(t)
	config.Serving.OpsAddr = reservedAddr(t)

	return config
}

func openTestVault(t *testing.T) *vault.Vault {
	t.Helper()

	opened, err := memoryvault.Open(1<<20, nil)
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	return opened
}

func reservedAddr(t *testing.T) string {
	t.Helper()

	var listen net.ListenConfig
	listener, err := listen.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release reserved port: %v", err)
	}

	return addr
}

type runningNode struct {
	peerAddr string
	opsAddr  string
	stop     context.CancelFunc
	stopped  chan error
}

func startNode(t *testing.T, config nodeconfiguration.Settings) runningNode {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	node := runningNode{
		peerAddr: config.Serving.PeerAddr,
		opsAddr:  config.Serving.OpsAddr,
		stop:     cancel,
		stopped:  make(chan error, 1),
	}
	storage := openTestVault(t)
	go func() {
		node.stopped <- yacynode.RunNode(ctx, config, storage, prometheus.NewRegistry())
	}()
	t.Cleanup(cancel)

	return node
}

func (n runningNode) wait(t *testing.T) error {
	t.Helper()

	select {
	case err := <-n.stopped:
		return err
	case <-time.After(settleFor):
		t.Fatal("RunNode neither stopped nor failed")

		return nil
	}
}

func (n runningNode) get(t *testing.T, path string) {
	t.Helper()

	n.awaitReady(t)
	n.answerTo(t, http.MethodGet, "http://"+n.peerAddr+path, nil)
}

func (n runningNode) hello(t *testing.T, req yacyproto.HelloRequest) yacyproto.HelloResponse {
	t.Helper()

	n.awaitReady(t)

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
		yacyproto.ParseMessage(body),
	)
	if err != nil {
		t.Fatalf("ParseHelloResponse: %v", err)
	}

	return parsed
}

func (n runningNode) metrics(t *testing.T) string {
	t.Helper()

	n.awaitReady(t)
	_, published := n.answerTo(t, http.MethodGet, "http://"+n.opsAddr+"/metrics", nil)

	return published
}

func (n runningNode) awaitReady(t *testing.T) {
	t.Helper()

	deadline := time.Now().Add(settleFor)
	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"http://"+n.opsAddr+"/metrics",
			nil,
		)
		if err != nil {
			t.Fatalf("build readiness request: %v", err)
		}
		if resp, err := http.DefaultClient.Do(req); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		select {
		case err := <-n.stopped:
			t.Fatalf("RunNode stopped before it served: %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatalf("ops endpoint never answered at %s", n.opsAddr)
}

func (n runningNode) answerTo(
	t *testing.T,
	method, address string,
	sent io.Reader,
) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, address, sent)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if sent != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, address, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	return resp.StatusCode, string(body)
}

func envFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}
