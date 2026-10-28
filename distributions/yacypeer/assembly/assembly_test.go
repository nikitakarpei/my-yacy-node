package yacypeerassembly_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
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
	return map[string]string{
		yacypeerassembly.EnvDataDirectory: dataDirectoryWithSolvedPeerKey(t),
		yacypeerassembly.EnvSeedlistURLs:  "http://127.0.0.1:1/seed.txt",
		"YACY_ADVERTISE_HOST":             "127.0.0.1",
		"YACYNODE__YACY_PEER_ADDR":        addressesBelowEphemeralPorts(t, 1)[0],
	}
}

func dataDirectoryWithSolvedPeerKey(t *testing.T) string {
	t.Helper()
	dataDirectory := t.TempDir()
	keyDirectory := filepath.Join(dataDirectory, "peer-key")
	if err := os.Mkdir(keyDirectory, 0o700); err != nil {
		t.Fatalf("make the peer key directory: %v", err)
	}
	keyText := []byte(peerkeytest.Key(0).String() + "\n")
	if err := os.WriteFile(filepath.Join(keyDirectory, "peer-key"), keyText, 0o600); err != nil {
		t.Fatalf("keep the peer key: %v", err)
	}
	return dataDirectory
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
	web, err := servergroup.Listen(t.Context(), anyLoopbackPort)
	if err != nil {
		t.Fatalf("listen for the web address: %v", err)
	}
	return cfg, web
}

func TestRunPeerServesTheMetricsOfEveryServiceUntilCancelled(t *testing.T) {
	env := peerEnvironment(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()

	awaitEveryService(t, env, web)
	awaitAnswer(t, "http://"+web.Addr().String()+"/metrics", "go_goroutines")

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerShowsTheDashboardsOfItsServicesFromTheirMetricHistory(t *testing.T) {
	env := peerEnvironment(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web)

	awaitAnswer(t, "http://"+web.Addr().String()+"/dashboards",
		"/dashboards/yacynode", "/dashboards/yacycrawler", "/dashboards/yacydhtsearch")
	since := time.Now().Add(-time.Hour).Unix()
	awaitAnswer(t, fmt.Sprintf(
		"http://%s/dashboards/api/v1/query_range?query=go_goroutines%%20and%%20yacynode_info&start=%d&end=%d&step=15",
		web.Addr(),
		since,
		since+2*3600,
	), `"instance":"yacynode"`)

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerSendsOutboundRequestsOnThroughTheEgressProxyTheOperatorNames(t *testing.T) {
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
	env[yacypeerassembly.EnvPeerProxyURL] = proxy.URL
	env[yacypeerassembly.EnvSeedlistURLs] = "http://203.0.113.5/seed.txt"
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()

	select {
	case proxiedURL := <-proxiedURLs:
		if proxiedURL != env[yacypeerassembly.EnvSeedlistURLs] {
			t.Errorf("proxied %q, want the seedlist", proxiedURL)
		}
	case <-time.After(startDeadline):
		t.Error("no request reached the egress proxy the operator names")
	}

	awaitEveryService(t, env, web)
	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerSearchesTheYaCyNetworkAndTheCrawledPagesOnTheSearchPage(t *testing.T) {
	env := peerEnvironment(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web)

	awaitAnswer(t, "http://"+web.Addr().String()+"/search?q=yacy",
		"<dt>!ct</dt>", "<dt>!ya</dt>", "No results.")

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerLinksTheSearchPageTheCrawlConsoleAndTheDashboardsInATabBar(t *testing.T) {
	env := peerEnvironment(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web)
	const (
		searchTab  = `href="/"`
		crawlsTab  = `href="/crawlconsole/"`
		metricsTab = `href="/dashboards"`
	)

	for path, currentTab := range map[string]string{
		"/":              searchTab,
		"/crawlconsole/": crawlsTab,
		"/dashboards":    metricsTab,
	} {
		awaitAnswer(t, "http://"+web.Addr().String()+path,
			searchTab, crawlsTab, metricsTab, currentTab+` aria-current="page"`)
	}

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerSendsAVisitFromTheSearchPageOnToThePageWithALinkThatThePeerSecretSigns(
	t *testing.T,
) {
	env := peerEnvironment(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web)

	secret, err := visitlinksecret.Settle(
		filepath.Join(env[yacypeerassembly.EnvDataDirectory], "visitcrawl"),
	)
	if err != nil {
		t.Fatalf("read the peer secret: %v", err)
	}
	const visitedPage = "https://example.org/visited"
	destination := redirectOf(t, "http://"+web.Addr().String()+"/visit?"+
		visitLinkQueryFor(visitedPage, secret))
	if destination != visitedPage {
		t.Errorf("visit redirected to %q, want %q", destination, visitedPage)
	}

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerServesEveryLinkOfItsWebUIsOnTheWebAddress(t *testing.T) {
	env := peerEnvironment(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web)

	for _, page := range []string{"/", "/search?q=yacy", "/crawlconsole/", "/crawlconsole/crawls/new", "/dashboards"} {
		body, _ := scraped("http://" + web.Addr().String() + page)
		links := localLinksOf(body)
		if len(links) == 0 {
			t.Errorf("%s links nothing", page)
		}
		for _, link := range links {
			if status := statusOf(t, "http://"+web.Addr().String()+link); status != http.StatusOK {
				t.Errorf("%s links %s, which answers %d", page, link, status)
			}
		}
	}

	cancel()
	awaitShutdown(t, runDone)
}

var localLinkPattern = regexp.MustCompile(`(?:href|src)="(/[^"]*)"`)

func localLinksOf(body string) []string {
	matches := localLinkPattern.FindAllStringSubmatch(body, -1)
	links := make([]string, 0, len(matches))
	for _, match := range matches {
		links = append(links, html.UnescapeString(match[1]))
	}
	return links
}

func statusOf(t *testing.T, pageURL string) int {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, pageURL, nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get %s: %v", pageURL, err)
	}
	_ = response.Body.Close()
	return response.StatusCode
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
		"SEARCHPAGE_LISTEN_ADDR":                 ":8097",
		"CRAWLCONSOLE_PATH_PREFIX":               "/crawls",
		"CRAWLCONSOLE_TAB_BAR":                   `{"tabs":[]}`,
		"SEARCHPAGE__SEARCHPAGE_TAB_BAR":         `{"tabs":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			env := peerEnvironment(t)
			env[name] = value
			cfg, web := peerFrom(t, env)

			err := yacypeerassembly.RunPeer(t.Context(), cfg, web, envFrom(env))
			if err == nil || !strings.Contains(err.Error(), name) {
				t.Errorf("RunPeer = %v, want the overridden wiring named", err)
			}
		})
	}
}

func TestRunPeerRequiresTheAdvertiseHostBehindTheEgressProxyTheOperatorNames(t *testing.T) {
	env := peerEnvironment(t)
	env[yacypeerassembly.EnvPeerProxyURL] = "http://proxy.example:3128"
	delete(env, "YACY_ADVERTISE_HOST")
	cfg, web := peerFrom(t, env)

	err := yacypeerassembly.RunPeer(t.Context(), cfg, web, envFrom(env))
	if err == nil || !strings.Contains(err.Error(), "YACY_ADVERTISE_HOST") {
		t.Errorf("RunPeer = %v, want the advertise host required", err)
	}
}

func TestRunPeerRefusesAServiceOpsAddress(t *testing.T) {
	env := peerEnvironment(t)
	env["CORPUSTEXT_OPS_ADDR"] = ":9096"
	cfg, web := peerFrom(t, env)

	err := yacypeerassembly.RunPeer(t.Context(), cfg, web, envFrom(env))
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT_OPS_ADDR") {
		t.Errorf("RunPeer = %v, want the service web address named", err)
	}
}

func TestRunPeerFailsWhenAServiceConfigIsInvalid(t *testing.T) {
	env := peerEnvironment(t)
	env["CORPUSTEXT__SEARCH_INDEX_ENGINE"] = "unknown"
	cfg, web := peerFrom(t, env)

	err := yacypeerassembly.RunPeer(t.Context(), cfg, web, envFrom(env))
	if err == nil || !strings.HasPrefix(err.Error(), "corpustext:") {
		t.Errorf("RunPeer = %v, want the corpustext config refused", err)
	}
}

func TestRunPeerReleasesItsListenAddressesWhenAServiceCannotListen(t *testing.T) {
	env := peerEnvironment(t)
	occupied, err := servergroup.Listen(t.Context(), env["YACYNODE__YACY_PEER_ADDR"])
	if err != nil {
		t.Fatalf("occupy the yacynode peer address: %v", err)
	}
	defer func() { _ = occupied.Close() }()
	cfg, web := peerFrom(t, env)

	if err := yacypeerassembly.RunPeer(t.Context(), cfg, web, envFrom(env)); err == nil {
		t.Fatal("RunPeer = nil, want the occupied yacynode peer address refused")
	}
	reopened, err := servergroup.Listen(t.Context(), web.Addr().String())
	if err != nil {
		t.Fatalf("listen on the web address after RunPeer failed: %v", err)
	}
	_ = reopened.Close()
}

func awaitEveryService(t *testing.T, env map[string]string, web net.Listener) {
	t.Helper()
	awaitAnswer(t, "http://"+env["YACYNODE__YACY_PEER_ADDR"]+"/")
	awaitAnswer(t, "http://"+web.Addr().String()+"/crawlconsole/", "No running crawls.")
	for _, service := range []string{
		"yacynode", "yacydhtsearch", "corpustext",
		"yacycrawler", "pagescrape", "crawlconsole", "crawlscrapebridge", "visitcrawl",
		"searchpage",
	} {
		awaitAnswer(t, "http://"+web.Addr().String()+"/metrics/"+service, service+"_info 1")
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
