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

	yacydhtsearch "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/cmd/yacydhtsearch"
)

func TestTheServiceAnswersSearchesAndPublishesMetricsUntilItStops(t *testing.T) {
	cfg := yacydhtsearch.ServiceConfig{
		ListenAddr:         reservedPort(t),
		OpsAddr:            reservedPort(t),
		NetworkName:        "freeworld",
		SeedlistURLs:       []string{"http://127.0.0.1:1/yacy/seedlist.html"},
		EgressProxyURL:     &url.URL{Scheme: "http", Host: "127.0.0.1:1"},
		QueryBudget:        time.Second,
		PeerChoiceCooldown: 5 * time.Second,
		NetworkRedundancy:  2,
		PeerCallsInFlight:  2,
		PeerCallBudget:     time.Second,
		ProbesInFlight:     2,
		DirectoryCapacity:  8,
		RefreshInterval:    time.Hour,
		ProbeBudget:        time.Second,
		Partitions:         16,
		MaxResponseBytes:   1024,
		RankedItemsCeiling: 50,

		PagesReadPerQuery:    10,
		PageReadBudget:       time.Second,
		PageByteCeiling:      1024,
		SnippetLengthCeiling: 300,
	}

	ctx, stop := context.WithCancel(t.Context())
	stopped := make(chan error, 1)
	go func() { stopped <- yacydhtsearch.RunService(ctx, cfg, prometheus.NewRegistry()) }()

	metrics := bodyOnceServed(t, "http://"+cfg.OpsAddr+"/metrics")
	if !strings.Contains(metrics, "yacydhtsearch_directory_capacity") {
		t.Fatalf("/metrics does not publish the directory capacity:\n%s", metrics)
	}

	answer := bodyOnceServed(t, "http://"+cfg.ListenAddr+"/yacysearch.json?query=berlin")
	if !strings.Contains(answer, `"channels"`) {
		t.Fatalf("/yacysearch.json = %q, want YaCy's public search form", answer)
	}

	stop()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatalf("RunService: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunService did not stop after the context was cancelled")
	}
}

func reservedPort(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	address := listener.Addr().String()
	_ = listener.Close()

	return address
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
