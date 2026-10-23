package yacypeerassembly_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
)

const (
	shutdownDeadline = 30 * time.Second
	startDeadline    = 10 * time.Second
	startPollPause   = 50 * time.Millisecond
)

func peerEnvironment(t *testing.T, peerAddress string) map[string]string {
	t.Helper()
	return map[string]string{
		yacypeerassembly.EnvDataDirectory:            t.TempDir(),
		yacypeerassembly.EnvSeedlistURLs:             "http://127.0.0.1:1/seed.txt",
		yacypeerassembly.EnvOpsAddr:                  "127.0.0.1:0",
		"EGRESS_PROXY_URL":                           "http://127.0.0.1:1",
		"YACY_ADVERTISE_HOST":                        "127.0.0.1",
		"YACYNODE__YACY_PEER_ADDR":                   peerAddress,
		"YACYDHTSEARCH__YACYDHTSEARCH_LISTEN_ADDR":   "127.0.0.1:0",
		"CORPUSTEXT__CORPUSTEXT_SEARCH_ADDR":         "127.0.0.1:0",
		"CORPUSMARKDOWN__CORPUSMARKDOWN_LISTEN_ADDR": "127.0.0.1:0",
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find a free address: %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String()
}

func TestRunServiceServesTheMetricsOfEveryMemberUntilCancelled(t *testing.T) {
	peerAddress := freeAddress(t)
	env := peerEnvironment(t, peerAddress)
	cfg, err := yacypeerassembly.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	listeners, err := yacypeerassembly.ListenersFor(t.Context(), cfg)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunService(ctx, cfg, listeners, envFrom(env))
	}()

	awaitAnswer(t, "http://"+peerAddress+"/")
	opsURL := "http://" + listeners.Ops.Addr().String()
	for _, member := range []string{"yacynode", "yacydhtsearch", "corpustext", "corpusmarkdown"} {
		awaitAnswer(t, opsURL+"/metrics/"+member, member+"_info 1")
	}

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunServiceRefusesWiringTheOperatorSets(t *testing.T) {
	env := peerEnvironment(t, freeAddress(t))
	env["CORPUSTEXT__SCRAPE_PAGE_OFFER_NATS_URL"] = "nats://elsewhere:4222"
	cfg, err := yacypeerassembly.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	listeners, err := yacypeerassembly.ListenersFor(t.Context(), cfg)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}

	err = yacypeerassembly.RunService(t.Context(), cfg, listeners, envFrom(env))
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT__SCRAPE_PAGE_OFFER_NATS_URL") {
		t.Errorf("RunService = %v, want the overridden wiring named", err)
	}
}

func TestRunServiceFailsWhenAMemberConfigIsInvalid(t *testing.T) {
	env := peerEnvironment(t, freeAddress(t))
	env["CORPUSTEXT__SEARCH_INDEX_ENGINE"] = "unknown"
	cfg, err := yacypeerassembly.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	listeners, err := yacypeerassembly.ListenersFor(t.Context(), cfg)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}

	err = yacypeerassembly.RunService(t.Context(), cfg, listeners, envFrom(env))
	if err == nil || !strings.HasPrefix(err.Error(), "corpustext:") {
		t.Errorf("RunService = %v, want the corpustext config refused", err)
	}
}

func TestListenersForFailsOnAnAddressInUse(t *testing.T) {
	cfg := yacypeerassembly.ServiceConfig{OpsAddr: "127.0.0.1:0"}
	listeners, err := yacypeerassembly.ListenersFor(t.Context(), cfg)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	defer func() { _ = listeners.Ops.Close() }()

	cfg.OpsAddr = listeners.Ops.Addr().String()
	if _, err := yacypeerassembly.ListenersFor(t.Context(), cfg); err == nil {
		t.Error("ListenersFor bound an address in use, want an error")
	}
}

func awaitAnswer(t *testing.T, url string, wantedText ...string) {
	t.Helper()
	deadline := time.Now().Add(startDeadline)
	for time.Now().Before(deadline) {
		if body, ok := scraped(url); ok && containsEvery(body, wantedText) {
			return
		}
		time.Sleep(startPollPause)
	}
	t.Fatalf("%s never answered with %q", url, wantedText)
}

func scraped(url string) (string, bool) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", false
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", false
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	return string(body), err == nil
}

func containsEvery(body string, wantedText []string) bool {
	for _, text := range wantedText {
		if !strings.Contains(body, text) {
			return false
		}
	}
	return true
}

func awaitShutdown(t *testing.T, runDone <-chan error) {
	t.Helper()
	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("RunService: %v", err)
		}
	case <-time.After(shutdownDeadline):
		t.Fatal("RunService did not stop after cancel")
	}
}
