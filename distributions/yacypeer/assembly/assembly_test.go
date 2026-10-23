package yacypeerassembly_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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
		"YACY_ADVERTISE_HOST":                        "127.0.0.1",
		"YACYNODE__YACY_PEER_ADDR":                   freeAddress(t),
		"YACYDHTSEARCH__YACYDHTSEARCH_LISTEN_ADDR":   anyLoopbackPort,
		"CORPUSTEXT__CORPUSTEXT_SEARCH_ADDR":         anyLoopbackPort,
		"CORPUSMARKDOWN__CORPUSMARKDOWN_LISTEN_ADDR": anyLoopbackPort,
		"CRAWLCONSOLE_LISTEN_ADDR":                   freeAddress(t),
	}
}

func peerFrom(t *testing.T, env map[string]string) (yacypeerassembly.PeerConfig, net.Listener) {
	t.Helper()
	cfg, err := yacypeerassembly.LoadPeerConfig(envFrom(env))
	if err != nil {
		t.Fatalf("load the config: %v", err)
	}
	ops, err := servergroup.Listen(t.Context(), anyLoopbackPort)
	if err != nil {
		t.Fatalf("listen for ops: %v", err)
	}
	return cfg, ops
}

func TestRunPeerServesTheMetricsOfEveryServiceUntilCancelled(t *testing.T) {
	env := peerEnvironment(t)
	cfg, ops := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, ops, envFrom(env))
	}()

	awaitEveryService(t, env, ops)
	awaitAnswer(t, "http://"+ops.Addr().String()+"/metrics", "go_goroutines")

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerSendsOutboundRequestsThroughTheEgressProxyTheOperatorNames(t *testing.T) {
	proxiedURLs := make(chan string, 16)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case proxiedURLs <- r.URL.String():
		default:
		}
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(proxy.Close)
	env := peerEnvironment(t)
	env[yacypeerassembly.EnvEgressProxyURL] = proxy.URL
	cfg, ops := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, ops, envFrom(env))
	}()

	select {
	case proxiedURL := <-proxiedURLs:
		if proxiedURL != env[yacypeerassembly.EnvSeedlistURLs] {
			t.Errorf("proxied %q, want the seedlist", proxiedURL)
		}
	case <-time.After(startDeadline):
		t.Error("no request reached the egress proxy the operator names")
	}

	awaitEveryService(t, env, ops)
	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerRefusesWiringTheOperatorSets(t *testing.T) {
	for name, value := range map[string]string{
		"CORPUSTEXT__SCRAPE_PAGE_OFFER_NATS_URL": "nats://elsewhere:4222",
		"EGRESS_PROXY_URL":                       "http://elsewhere:4750",
	} {
		t.Run(name, func(t *testing.T) {
			env := peerEnvironment(t)
			env[name] = value
			cfg, ops := peerFrom(t, env)

			err := yacypeerassembly.RunPeer(t.Context(), cfg, ops, envFrom(env))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("RunPeer = %v, want the overridden wiring named", err)
			}
		})
	}
}

func TestRunPeerRefusesAServiceOpsAddress(t *testing.T) {
	env := peerEnvironment(t)
	env["CORPUSTEXT_OPS_ADDR"] = ":9096"
	cfg, ops := peerFrom(t, env)

	err := yacypeerassembly.RunPeer(t.Context(), cfg, ops, envFrom(env))
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT_OPS_ADDR") {
		t.Errorf("RunPeer = %v, want the service ops address named", err)
	}
}

func TestRunPeerFailsWhenAServiceConfigIsInvalid(t *testing.T) {
	env := peerEnvironment(t)
	env["CORPUSTEXT__SEARCH_INDEX_ENGINE"] = "unknown"
	cfg, ops := peerFrom(t, env)

	err := yacypeerassembly.RunPeer(t.Context(), cfg, ops, envFrom(env))
	if err == nil || !strings.HasPrefix(err.Error(), "corpustext:") {
		t.Errorf("RunPeer = %v, want the corpustext config refused", err)
	}
}

func awaitEveryService(t *testing.T, env map[string]string, ops net.Listener) {
	t.Helper()
	awaitAnswer(t, "http://"+env["YACYNODE__YACY_PEER_ADDR"]+"/")
	awaitAnswer(t, "http://"+env["CRAWLCONSOLE_LISTEN_ADDR"]+"/", "No running crawls.")
	for _, service := range []string{
		"yacynode", "yacydhtsearch", "corpustext", "corpusmarkdown",
		"yacycrawler", "pagescrape", "crawlconsole", "crawlscrapebridge",
	} {
		awaitAnswer(t, "http://"+ops.Addr().String()+"/metrics/"+service, service+"_info 1")
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
			t.Errorf("RunPeer: %v", err)
		}
	case <-time.After(shutdownDeadline):
		t.Fatal("RunPeer did not stop after cancel")
	}
}
