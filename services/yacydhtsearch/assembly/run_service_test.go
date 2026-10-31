package yacydhtsearchassembly_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	yacydhtsearch "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestTheServiceAnswersSearchesAndPublishesMetricsUntilItStops(t *testing.T) {
	cfg := serviceConfigFor(t)
	listeners := yacydhtsearch.Listeners{Search: listenerFor(t)}
	searchAddr := listeners.Search.Addr().String()
	registry := prometheus.NewRegistry()

	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- yacydhtsearch.RunService(ctx, cfg, listeners, registry) }()

	answer := bodyOnceServed(t, "http://"+searchAddr+"/yacysearch.json?query=berlin")
	if !strings.Contains(answer, `"channels"`) {
		t.Fatalf("/yacysearch.json = %q, want YaCy's public search form", answer)
	}

	metrics := metricNamesOf(t, registry)
	for _, want := range []string{"yacydhtsearch_info", "yacydhtsearch_directory_capacity"} {
		if !metrics[want] {
			t.Errorf("the registry does not publish %s", want)
		}
	}

	stop()
	awaitStop(t, stopped)
}

func TestTheOpsAddressServesMetricsButNoProfilerByDefault(t *testing.T) {
	cfg := serviceConfigFor(t)
	listeners := yacydhtsearch.Listeners{Search: listenerFor(t)}
	ops := listenerFor(t)
	opsAddr := ops.Addr().String()
	registry := prometheus.NewRegistry()

	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() {
		stopped <- opsmetrics.ServeAlongside(ctx, ops, yacydhtsearch.OpsMuxFor(cfg, registry),
			func(ctx context.Context) error {
				return yacydhtsearch.RunService(ctx, cfg, listeners, registry)
			})
	}()

	awaitServing(t, listeners)
	metrics := bodyOnceServed(t, "http://"+opsAddr+"/metrics")
	if !strings.Contains(metrics, "yacydhtsearch_info 1") {
		t.Fatalf("/metrics does not publish the service identity:\n%s", metrics)
	}
	if status := statusOf(t, "http://"+opsAddr+"/debug/pprof/"); status != http.StatusNotFound {
		t.Fatalf(
			"/debug/pprof/ = %d, want %d while the profiler is off",
			status,
			http.StatusNotFound,
		)
	}

	stop()
	awaitStop(t, stopped)
}

func metricNamesOf(t *testing.T, registry *prometheus.Registry) map[string]bool {
	t.Helper()

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	names := map[string]bool{}
	for _, family := range families {
		names[family.GetName()] = true
	}

	return names
}

func serviceConfigFor(t *testing.T) yacydhtsearch.ServiceConfig {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	return yacydhtsearch.ServiceConfig{
		NetworkName:                    "freeworld",
		SeedlistURLs:                   []string{"http://127.0.0.1:1/yacy/seedlist.html"},
		SeedlistReadBudget:             time.Second,
		EgressProxyURL:                 &url.URL{Scheme: "http", Host: "127.0.0.1:1"},
		QueryBudget:                    time.Second,
		NetworkRedundancy:              2,
		ReplicasCoveringAPartition:     2,
		HedgeDelay:                     100 * time.Millisecond,
		PeerCallsInFlight:              2,
		URLMetadataCallBudget:          time.Second,
		SearchCallBudget:               time.Second,
		ProbesInFlight:                 2,
		DirectoryCapacity:              8,
		RefreshInterval:                time.Hour,
		ProbeBudget:                    time.Second,
		Partitions:                     partitions,
		MaxResponseBytes:               1024,
		RankedItemsCeiling:             50,
		URLMetadataAskDocumentsCeiling: 1000,
		URLMetadataAskDocumentsFloor:   25,
		URLMetadataAsksPerDocument:     2,
		URLMetadataAskTargetTime:       time.Second,

		PagesReadPerQueryCeiling: 10,
		PagesReadPerSiteCeiling:  10,
		PageReadBudget:           time.Second,
		PageByteCeiling:          1024,
		PageReadMaxRedirectHops:  3,
		SnippetLengthCeiling:     300,
	}
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

func bodyOnceServed(t *testing.T, address string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		t.Fatalf("build request for %s: %v", address, err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s never answered: %v", address, err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%s answered %d, want %d", address, response.StatusCode, http.StatusOK)
	}

	return string(body)
}

func awaitServing(t *testing.T, listeners yacydhtsearch.Listeners) {
	t.Helper()
	statusOf(t, "http://"+listeners.Search.Addr().String()+"/")
}

func statusOf(t *testing.T, address string) int {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatalf("build request for %s: %v", address, err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", address, err)
	}
	_ = response.Body.Close()

	return response.StatusCode
}

func awaitStop(t *testing.T, stopped <-chan error) {
	t.Helper()

	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("stopped with: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the service did not stop after the context was cancelled")
	}
}

func TestTheOpsAddressServesTheProfilerWhenTheOperatorOptsIn(t *testing.T) {
	cfg := serviceConfigFor(t)
	cfg.ServeProfiler = true
	listeners := yacydhtsearch.Listeners{Search: listenerFor(t)}
	ops := listenerFor(t)
	opsAddr := ops.Addr().String()
	registry := prometheus.NewRegistry()

	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() {
		stopped <- opsmetrics.ServeAlongside(ctx, ops, yacydhtsearch.OpsMuxFor(cfg, registry),
			func(ctx context.Context) error {
				return yacydhtsearch.RunService(ctx, cfg, listeners, registry)
			})
	}()

	profiles := bodyOnceServed(t, "http://"+opsAddr+"/debug/pprof/")
	if !strings.Contains(profiles, "goroutine") {
		t.Fatalf("/debug/pprof/ does not list the goroutine profile:\n%s", profiles)
	}
	awaitServing(t, listeners)
	metrics := bodyOnceServed(t, "http://"+opsAddr+"/metrics")
	if !strings.Contains(metrics, "yacydhtsearch_info 1") {
		t.Fatalf("/metrics does not publish the service identity:\n%s", metrics)
	}

	stop()
	awaitStop(t, stopped)
}

func TestTheServiceRefusesToStartWhenItsStateServerDoesNotAnswer(t *testing.T) {
	cfg := serviceConfigFor(t)
	cfg.StateRedis = &redis.Options{Addr: "127.0.0.1:1"}
	listeners := yacydhtsearch.Listeners{Search: listenerFor(t)}

	if err := yacydhtsearch.RunService(
		t.Context(),
		cfg,
		listeners,
		prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("RunService started with a Redis address that does not answer")
	}
}

func TestListenersForListensOnTheSearchAddress(t *testing.T) {
	listeners, err := yacydhtsearch.ListenersFor(
		t.Context(), yacydhtsearch.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Search.Close() })
	if listeners.Search.Addr().String() == "127.0.0.1:0" {
		t.Errorf("search listener = %s, want a bound port", listeners.Search.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := yacydhtsearch.ListenersFor(
		t.Context(), yacydhtsearch.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable search address")
	}
}

func TestListenersCloseReleasesTheSearchAddress(t *testing.T) {
	listeners, err := yacydhtsearch.ListenersFor(
		t.Context(), yacydhtsearch.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}

	if err := listeners.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := servergroup.Listen(t.Context(), listeners.Search.Addr().String())
	if err != nil {
		t.Fatalf("listen on the closed search address: %v", err)
	}
	_ = reopened.Close()
}
