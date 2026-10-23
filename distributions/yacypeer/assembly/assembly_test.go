package yacypeerassembly_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
)

const (
	shutdownDeadline = 30 * time.Second
	startDeadline    = 10 * time.Second
	startPollPause   = 50 * time.Millisecond
	anyLoopbackPort  = "127.0.0.1:0"
)

func peerEnvironment(t *testing.T) map[string]string {
	t.Helper()
	return map[string]string{
		yacypeerassembly.EnvDataDirectory:            t.TempDir(),
		yacypeerassembly.EnvSeedlistURLs:             "http://127.0.0.1:1/seed.txt",
		"EGRESS_PROXY_URL":                           "http://127.0.0.1:1",
		"YACY_ADVERTISE_HOST":                        "127.0.0.1",
		"YACYNODE__YACY_PEER_ADDR":                   freeAddress(t),
		"YACYDHTSEARCH__YACYDHTSEARCH_LISTEN_ADDR":   anyLoopbackPort,
		"CORPUSTEXT__CORPUSTEXT_SEARCH_ADDR":         anyLoopbackPort,
		"CORPUSMARKDOWN__CORPUSMARKDOWN_LISTEN_ADDR": anyLoopbackPort,
	}
}

func peerFrom(t *testing.T, env map[string]string) (yacypeerassembly.ServiceConfig, net.Listener) {
	t.Helper()
	cfg, err := yacypeerassembly.LoadServiceConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	ops, err := servergroup.Listen(t.Context(), anyLoopbackPort)
	if err != nil {
		t.Fatalf("listen for ops: %v", err)
	}
	return cfg, ops
}

func TestRunServiceServesTheMetricsOfEveryMemberUntilCancelled(t *testing.T) {
	env := peerEnvironment(t)
	cfg, ops := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunService(ctx, cfg, ops, envFrom(env))
	}()

	awaitAnswer(t, "http://"+env["YACYNODE__YACY_PEER_ADDR"]+"/")
	opsURL := "http://" + ops.Addr().String()
	awaitAnswer(t, opsURL+"/metrics", "go_goroutines")
	for _, member := range []string{"yacynode", "yacydhtsearch", "corpustext", "corpusmarkdown"} {
		awaitAnswer(t, opsURL+"/metrics/"+member, member+"_info 1")
	}

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunServiceRefusesWiringTheOperatorSets(t *testing.T) {
	env := peerEnvironment(t)
	env["CORPUSTEXT__SCRAPE_PAGE_OFFER_NATS_URL"] = "nats://elsewhere:4222"
	cfg, ops := peerFrom(t, env)

	err := yacypeerassembly.RunService(t.Context(), cfg, ops, envFrom(env))
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT__SCRAPE_PAGE_OFFER_NATS_URL") {
		t.Errorf("RunService = %v, want the overridden wiring named", err)
	}
}

func TestRunServiceRefusesAMemberOpsAddress(t *testing.T) {
	env := peerEnvironment(t)
	env["CORPUSTEXT_OPS_ADDR"] = ":9096"
	cfg, ops := peerFrom(t, env)

	err := yacypeerassembly.RunService(t.Context(), cfg, ops, envFrom(env))
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT_OPS_ADDR") {
		t.Errorf("RunService = %v, want the member ops address named", err)
	}
}

func TestRunServiceFailsWhenAMemberConfigIsInvalid(t *testing.T) {
	env := peerEnvironment(t)
	env["CORPUSTEXT__SEARCH_INDEX_ENGINE"] = "unknown"
	cfg, ops := peerFrom(t, env)

	err := yacypeerassembly.RunService(t.Context(), cfg, ops, envFrom(env))
	if err == nil || !strings.HasPrefix(err.Error(), "corpustext:") {
		t.Errorf("RunService = %v, want the corpustext config refused", err)
	}
}

func freeAddress(t *testing.T) string {
	t.Helper()
	listener, err := servergroup.Listen(t.Context(), anyLoopbackPort)
	if err != nil {
		t.Fatalf("find a free address: %v", err)
	}
	defer func() { _ = listener.Close() }()
	return listener.Addr().String()
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
