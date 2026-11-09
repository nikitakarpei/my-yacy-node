package yacypeerassembly_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
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

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	yacypeerassembly "github.com/nikitakarpei/yacy-rwi-node/yacypeer/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess/companionaccesstest"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/paireddevices"
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
		"YACYNODE__YACY_PEER_ADDR":        addressBelowEphemeralPorts(t),
	}
}

func dataDirectoryWithSolvedPeerKey(t *testing.T) string {
	t.Helper()
	dataDirectory := t.TempDir()
	keyText := []byte(peerkeytest.SecretTextOf(peerkeytest.Key(0)) + "\n")
	if err := os.WriteFile(filepath.Join(dataDirectory, "peer.key"), keyText, 0o400); err != nil {
		t.Fatalf("keep the peer key: %v", err)
	}
	return dataDirectory
}

func addressBelowEphemeralPorts(t *testing.T) string {
	t.Helper()
	for range portAttempts {
		port := firstListenPort + rand.IntN(listenPortSpan) //nolint:gosec // G404: no secrecy.
		listener, err := servergroup.Listen(
			t.Context(), net.JoinHostPort(loopbackHost, strconv.Itoa(port)),
		)
		if err == nil {
			address := listener.Addr().String()
			_ = listener.Close()
			return address
		}
	}
	t.Fatalf("no free port below %d", firstEphemeralPort)
	return ""
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

	awaitEveryService(t, env, web, runDone)
	awaitAnswer(t, runDone, "http://"+web.Addr().String()+"/metrics", "go_goroutines")

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
	awaitEveryService(t, env, web, runDone)

	awaitAnswer(t, runDone, "http://"+web.Addr().String()+"/dashboards",
		"/dashboards/yacynode", "/dashboards/yacycrawler", "/dashboards/yacydhtsearch")
	since := time.Now().Add(-time.Hour).Unix()
	awaitAnswer(t, runDone, fmt.Sprintf(
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

	awaitEveryService(t, env, web, runDone)
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
	awaitEveryService(t, env, web, runDone)

	awaitAnswer(t, runDone, "http://"+web.Addr().String()+"/search?q=yacy",
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
	awaitEveryService(t, env, web, runDone)
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
		awaitAnswer(t, runDone, "http://"+web.Addr().String()+path,
			searchTab, crawlsTab, metricsTab, currentTab+` aria-current="page"`)
		if body, _ := scraped("http://" + web.Addr().String() + path); strings.Contains(
			body, `href="/companions"`,
		) {
			t.Errorf("%s links the Companions pages of a peer without a companion address", path)
		}
	}

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunPeerWithACompanionAddressShowsTheCompanionsTabAndRefusesUnknownCompanions(
	t *testing.T,
) {
	env := peerEnvironment(t)
	env[yacypeerassembly.EnvCompanionAddr] = addressBelowEphemeralPorts(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web, runDone)
	const companionsTab = `href="/companions"`

	for _, path := range []string{"/", "/crawlconsole/", "/dashboards"} {
		awaitAnswer(t, runDone, "http://"+web.Addr().String()+path, companionsTab)
	}
	awaitAnswer(t, runDone, "http://"+web.Addr().String()+"/companions",
		companionsTab+` aria-current="page"`, "No companions are paired.")
	requireUnknownCompanionRefused(t, env[yacypeerassembly.EnvCompanionAddr])

	cancel()
	awaitShutdown(t, runDone)
}

func requireUnknownCompanionRefused(t *testing.T, companionAddress string) {
	t.Helper()
	connection, err := nats.Connect(
		"tls://"+companionAddress,
		nats.Secure(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // the test pins no broker key.
			MinVersion:         tls.VersionTLS13,
		}),
		nats.MaxReconnects(0),
	)
	if err == nil {
		connection.Close()
		t.Fatal("a companion without a key or a secret connected")
	}
	if !errors.Is(err, nats.ErrAuthorization) {
		t.Errorf("connect: %v, want the companion refused over TLS", err)
	}
}

func TestRunPeerTellsAPairedDeviceThePartitionExponentThatTheOperatorSets(t *testing.T) {
	env := peerEnvironment(t)
	env[yacypeerassembly.EnvCompanionAddr] = addressBelowEphemeralPorts(t)
	env[yacypeerassembly.EnvPartitionExponent] = "6"
	certificate, deviceID := companionaccesstest.DeviceCertificate(t)
	keepPairedDevice(t, env[yacypeerassembly.EnvDataDirectory], deviceID)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web, runDone)

	settings := settingsAskedBy(t, env[yacypeerassembly.EnvCompanionAddr], certificate, deviceID)

	if settings.PartitionExponent != 6 {
		t.Errorf("partition exponent = %d, want 6", settings.PartitionExponent)
	}
	cancel()
	awaitShutdown(t, runDone)
}

func keepPairedDevice(t *testing.T, dataDirectory string, deviceID companionprotocol.DeviceID) {
	t.Helper()
	devices, err := paireddevices.Open(filepath.Join(dataDirectory, "companions"))
	if err != nil {
		t.Fatalf("open the paired devices: %v", err)
	}
	defer devices.Close()
	devices.Add(t.Context(), paireddevices.Device{ID: deviceID, Name: "Pixel 8"})
}

func settingsAskedBy(
	t *testing.T,
	companionAddress string,
	certificate tls.Certificate,
	deviceID companionprotocol.DeviceID,
) companionprotocol.DeviceSettings {
	t.Helper()
	connection, err := nats.Connect(
		"tls://"+companionAddress,
		nats.Secure(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // the test pins no broker key.
			Certificates:       []tls.Certificate{certificate},
			MinVersion:         tls.VersionTLS13,
		}),
		nats.CustomInboxPrefix(companionprotocol.DeviceInboxPrefixOf(deviceID)),
		nats.MaxReconnects(0),
	)
	if err != nil {
		t.Fatalf("connect as the paired device: %v", err)
	}
	defer connection.Close()
	reply, err := connection.Request(
		companionprotocol.SettingsSubjectOf(deviceID), nil, startDeadline,
	)
	for deadline := time.Now().Add(startDeadline); errors.Is(err, nats.ErrNoResponders) &&
		time.Now().Before(deadline); {
		time.Sleep(startPollPause)
		reply, err = connection.Request(
			companionprotocol.SettingsSubjectOf(deviceID), nil, startDeadline,
		)
	}
	if err != nil {
		t.Fatalf("ask for the settings: %v", err)
	}
	var settings companionprotocol.DeviceSettings
	if err := json.Unmarshal(reply.Data, &settings); err != nil {
		t.Fatalf("read the settings: %v", err)
	}
	return settings
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
	awaitEveryService(t, env, web, runDone)

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
	env[yacypeerassembly.EnvCompanionAddr] = addressBelowEphemeralPorts(t)
	cfg, web := peerFrom(t, env)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- yacypeerassembly.RunPeer(ctx, cfg, web, envFrom(env))
	}()
	awaitEveryService(t, env, web, runDone)

	for _, page := range []string{
		"/", "/search?q=yacy", "/crawlconsole/", "/crawlconsole/crawls/new", "/dashboards",
		"/companions", "/companions/pair",
	} {
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
		"PAGE_RWI_PARTITION_EXPONENT":            "5",
		"YACYDHTSEARCH_PARTITION_EXPONENT":       "5",
		"YACY_DISTRIBUTION_PARTITION_EXPONENT":   "5",
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

func awaitEveryService(
	t *testing.T,
	env map[string]string,
	web net.Listener,
	runDone <-chan error,
) {
	t.Helper()
	awaitAnswer(t, runDone, "http://"+env["YACYNODE__YACY_PEER_ADDR"]+"/")
	awaitAnswer(t, runDone, "http://"+web.Addr().String()+"/crawlconsole/", "No running crawls.")
	for _, service := range []string{
		"yacynode", "yacydhtsearch", "corpustext",
		"yacycrawler", "pagescrape", "crawlconsole", "crawlscrapebridge", "visitcrawl",
		"searchpage",
	} {
		awaitAnswer(
			t,
			runDone,
			"http://"+web.Addr().String()+"/metrics/"+service,
			service+"_info 1",
		)
	}
}

func awaitAnswer(t *testing.T, runDone <-chan error, url string, wantedText ...string) {
	t.Helper()
	deadline := time.Now().Add(startDeadline)
	for time.Now().Before(deadline) {
		if body, ok := scraped(url); ok && containsEvery(body, wantedText) {
			return
		}
		select {
		case err := <-runDone:
			t.Fatalf("RunPeer stopped before %s answered: %v", url, err)
		case <-time.After(startPollPause):
		}
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
	return string(body), err == nil && response.StatusCode == http.StatusOK
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
