package opsmetrics_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/opsmetrics"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const serviceDeadline = 5 * time.Second

func TestProcessRegistryGathersRuntimeMetrics(t *testing.T) {
	families, err := opsmetrics.ProcessRegistry().Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	names := map[string]bool{}
	for _, family := range families {
		names[family.GetName()] = true
	}
	for _, want := range []string{"go_goroutines", "process_cpu_seconds_total"} {
		if !names[want] {
			t.Errorf("registry lacks %s", want)
		}
	}
}

func TestServeAlongsideServesTheRegistryUntilTheServiceReturns(t *testing.T) {
	listener, err := servergroup.Listen(context.Background(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "service_info",
		Help: "Service identity.",
	}, func() float64 { return 1 }))
	serviceStopped := errors.New("service stopped")
	scraped := make(chan string, 1)
	service := func(context.Context) error {
		scraped <- scrape(t, "http://"+listener.Addr().String()+"/metrics")
		return serviceStopped
	}

	done := make(chan error, 1)
	go func() {
		done <- opsmetrics.ServeAlongside(
			context.Background(), listener, opsmetrics.MuxFor(registry), service,
		)
	}()

	if body := <-scraped; !strings.Contains(body, "service_info 1") {
		t.Errorf("metrics = %q, want service_info", body)
	}
	if err := awaitReturn(t, done); !errors.Is(err, serviceStopped) {
		t.Errorf("ServeAlongside = %v, want %v", err, serviceStopped)
	}
}

func scrape(t *testing.T, url string) string {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("request: %v", err)
		return ""
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("scrape: %v", err)
		return ""
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Errorf("read: %v", err)
	}
	return string(body)
}

func awaitReturn(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(serviceDeadline):
		t.Fatal("ServeAlongside did not return")
		return nil
	}
}
