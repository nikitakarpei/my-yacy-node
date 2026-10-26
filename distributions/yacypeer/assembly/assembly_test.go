package yacypeerassembly_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/visitlinksecret"
)

const (
	shutdownDeadline   = 30 * time.Second
	startDeadline      = 10 * time.Second
	startPollPause     = 50 * time.Millisecond
	anyLoopbackPort    = "127.0.0.1:0"
	loopbackHost       = "127.0.0.1"
	firstListenPort    = 10000
	firstEphemeralPort = 32768
	listenPortSpan     = firstEphemeralPort - firstListenPort
	portAttempts       = 1000
)

func peerEnvironment(t *testing.T) map[string]string {
	t.Helper()
	listenAddresses := addressesBelowEphemeralPorts(t, 3)
	return map[string]string{
		yacypeerassembly.EnvDataDirectory:          t.TempDir(),
		yacypeerassembly.EnvSeedlistURLs:           "http://127.0.0.1:1/seed.txt",
		"YACY_ADVERTISE_HOST":                      "127.0.0.1",
		"YACYNODE__YACY_PEER_ADDR":                 listenAddresses[0],
		"YACYDHTSEARCH__YACYDHTSEARCH_LISTEN_ADDR": anyLoopbackPort,
		"CORPUSTEXT__CORPUSTEXT_SEARCH_ADDR":       anyLoopbackPort,
		"CRAWLCONSOLE_LISTEN_ADDR":                 listenAddresses[1],
		"VISITCRAWL_LISTEN_ADDR":                   listenAddresses[2],
	}
}

func addressesBelowEphemeralPorts(t *testing.T, count int) []string {
	t.Helper()
	listeners := make([]net.Listener, 0, count)
	defer func() {
		for _, listener := range listeners {
			_ = listener.Close()
		}
	}()
	for attempt := 0; len(listeners) < count; attempt++ {
		if attempt == portAttempts {
			t.Fatalf("no %d free ports below %d", count, firstEphemeralPort)
		}
		port := firstListenPort + rand.IntN(listenPortSpan) //nolint:gosec // G404: no secrecy.
		address := net.JoinHostPort(loopbackHost, strconv.Itoa(port))
		listener, err := servergroup.Listen(t.Context(), address)
		if err == nil {
			listeners = append(listeners, listener)
		}
	}
	addresses := make([]string, 0, count)
	for _, listener := range listeners {
		addresses = append(addresses, listener.Addr().String())
	}
	return addresses
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

func TestRunPeerShowsTheDashboardsOfItsServicesFromTheirMetricHistory(t *testing.T) {
	env := peerEnvironment(t)
	cfg, ops := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, ops, envFrom(env))
	}()
	awaitEveryService(t, env, ops)

	awaitAnswer(t, "http://"+ops.Addr().String()+"/dashboards",
		"/dashboards/yacynode", "/dashboards/yacycrawler", "/dashboards/yacydhtsearch")
	since := time.Now().Add(-time.Hour).Unix()
	awaitAnswer(t, fmt.Sprintf(
		"http://%s/dashboards/api/v1/query_range?query=go_goroutines%%20and%%20yacynode_info&start=%d&end=%d&step=15",
		ops.Addr(),
		since,
		since+2*3600,
	), `"instance":"yacynode"`)

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

func TestRunPeerSendsAVisitOnToThePageWithALinkThatThePeerSecretSigns(t *testing.T) {
	env := peerEnvironment(t)
	cfg, ops := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, ops, envFrom(env))
	}()
	awaitEveryService(t, env, ops)

	secret, err := visitlinksecret.Settle(
		filepath.Join(env[yacypeerassembly.EnvDataDirectory], "visitcrawl"),
	)
	if err != nil {
		t.Fatalf("read the peer secret: %v", err)
	}
	const visitedPage = "https://example.org/visited"
	destination := redirectOf(t, "http://"+env["VISITCRAWL_LISTEN_ADDR"]+"/visit?"+
		visitLinkQueryFor(visitedPage, secret))
	if destination != visitedPage {
		t.Errorf("visit redirected to %q, want %q", destination, visitedPage)
	}

	cancel()
	awaitShutdown(t, runDone)
}

func visitLinkQueryFor(visitedPage string, secret string) string {
	expires := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10)
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(expires + "\n" + visitedPage))
	return url.Values{
		"url":       {visitedPage},
		"expires":   {expires},
		"signature": {hex.EncodeToString(seal.Sum(nil))},
	}.Encode()
}

func redirectOf(t *testing.T, visitURL string) string {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, visitURL, nil)
	if err != nil {
		t.Fatalf("build the visit: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	return response.Header.Get("Location")
}

func TestRunPeerRefusesWiringTheOperatorSets(t *testing.T) {
	for name, value := range map[string]string{
		"CORPUSTEXT__SCRAPE_PAGE_OFFER_NATS_URL": "nats://elsewhere:4222",
		"EGRESS_PROXY_URL":                       "http://elsewhere:4750",
		"YACYCRAWLER_API_ADDR":                   "127.0.0.1:9100",
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

func TestRunPeerRequiresTheAdvertiseHostBehindTheEgressProxyTheOperatorNames(t *testing.T) {
	env := peerEnvironment(t)
	env[yacypeerassembly.EnvEgressProxyURL] = "http://proxy.example:3128"
	delete(env, "YACY_ADVERTISE_HOST")
	cfg, ops := peerFrom(t, env)

	err := yacypeerassembly.RunPeer(t.Context(), cfg, ops, envFrom(env))
	if err == nil || !strings.Contains(err.Error(), "YACY_ADVERTISE_HOST") {
		t.Errorf("RunPeer = %v, want the advertise host required", err)
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

func TestRunPeerReleasesItsListenAddressesWhenAServiceCannotListen(t *testing.T) {
	env := peerEnvironment(t)
	occupied, err := servergroup.Listen(t.Context(), env["VISITCRAWL_LISTEN_ADDR"])
	if err != nil {
		t.Fatalf("occupy the visitcrawl address: %v", err)
	}
	defer func() { _ = occupied.Close() }()
	cfg, ops := peerFrom(t, env)

	if err := yacypeerassembly.RunPeer(t.Context(), cfg, ops, envFrom(env)); err == nil {
		t.Fatal("RunPeer = nil, want the occupied visitcrawl address refused")
	}
	for _, address := range []string{
		ops.Addr().String(), env["YACYNODE__YACY_PEER_ADDR"], env["CRAWLCONSOLE_LISTEN_ADDR"],
	} {
		reopened, err := servergroup.Listen(t.Context(), address)
		if err != nil {
			t.Errorf("listen on %s after RunPeer failed: %v", address, err)
			continue
		}
		_ = reopened.Close()
	}
}

func awaitEveryService(t *testing.T, env map[string]string, ops net.Listener) {
	t.Helper()
	awaitAnswer(t, "http://"+env["YACYNODE__YACY_PEER_ADDR"]+"/")
	awaitAnswer(t, "http://"+env["CRAWLCONSOLE_LISTEN_ADDR"]+"/", "No running crawls.")
	for _, service := range []string{
		"yacynode", "yacydhtsearch", "corpustext",
		"yacycrawler", "pagescrape", "crawlconsole", "crawlscrapebridge", "visitcrawl",
	} {
		awaitAnswer(t, "http://"+ops.Addr().String()+"/metrics/"+service, service+"_info 1")
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
			t.Errorf("RunPeer: %v", err)
		}
	case <-time.After(shutdownDeadline):
		t.Fatal("RunPeer did not stop after cancel")
	}
}
