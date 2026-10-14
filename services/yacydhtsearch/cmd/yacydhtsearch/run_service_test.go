package main_test

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

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	yacydhtsearch "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/cmd/yacydhtsearch"
)

func TestTheServiceAnswersSearchesAndPublishesMetricsUntilItStops(t *testing.T) {
	cfg := serviceConfigFor(t)
	listeners := yacydhtsearch.Listeners{Search: listenerFor(t), Ops: listenerFor(t)}
	searchAddr := listeners.Search.Addr().String()
	opsAddr := listeners.Ops.Addr().String()

	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- yacydhtsearch.RunService(ctx, cfg, listeners, prometheus.NewRegistry()) }()

	metrics := bodyOnceServed(t, "http://"+opsAddr+"/metrics")
	if !strings.Contains(metrics, "yacydhtsearch_directory_capacity") {
		t.Fatalf("/metrics does not publish the directory capacity:\n%s", metrics)
	}

	answer := bodyOnceServed(t, "http://"+searchAddr+"/yacysearch.json?query=berlin")
	if !strings.Contains(answer, `"channels"`) {
		t.Fatalf("/yacysearch.json = %q, want YaCy's public search form", answer)
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

func serviceConfigFor(t *testing.T) yacydhtsearch.ServiceConfig {
	t.Helper()

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
		Partitions:                     16,
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

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
		if err != nil {
			t.Fatalf("build request for %s: %v", address, err)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		body, _ := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if response.StatusCode == http.StatusOK {
			return string(body)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s never answered", address)

	return ""
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
			t.Fatalf("RunService: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunService did not stop after the context was cancelled")
	}
}

func TestTheOpsAddressServesTheProfilerWhenTheOperatorOptsIn(t *testing.T) {
	cfg := serviceConfigFor(t)
	cfg.ServeProfiler = true
	listeners := yacydhtsearch.Listeners{Search: listenerFor(t), Ops: listenerFor(t)}

	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- yacydhtsearch.RunService(ctx, cfg, listeners, prometheus.NewRegistry()) }()

	profiles := bodyOnceServed(t, "http://"+listeners.Ops.Addr().String()+"/debug/pprof/")
	if !strings.Contains(profiles, "goroutine") {
		t.Fatalf("/debug/pprof/ does not list the goroutine profile:\n%s", profiles)
	}

	stop()
	awaitStop(t, stopped)
}

func TestTheServiceRefusesToStartWhenItsStateServerDoesNotAnswer(t *testing.T) {
	cfg := serviceConfigFor(t)
	cfg.StateRedis = &redis.Options{Addr: "127.0.0.1:1"}
	listeners := yacydhtsearch.Listeners{Search: listenerFor(t), Ops: listenerFor(t)}

	if err := yacydhtsearch.RunService(
		t.Context(),
		cfg,
		listeners,
		prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("RunService started with a Redis address that does not answer")
	}
}
