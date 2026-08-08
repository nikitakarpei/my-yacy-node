package main

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/progressobservers/prometheus"
)

func publishedPageStreams() []PageStreamConfig {
	streams := make([]PageStreamConfig, 0, len(pageRepresentationCatalog()))
	for _, preset := range pageRepresentationCatalog() {
		streams = append(streams, PageStreamConfig{
			Representation: preset.representation,
			Subject:        yacycrawlcontract.CrawledPageSubject(preset.representation),
			MaxMsgs:        DefaultMaxMsgs,
			Published:      preset.representation == yacycrawlcontract.PageRepresentationKindRWI,
		})
	}
	return streams
}

func TestRunServiceProcessesOrderThenStops(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	cfg := ServiceConfig{
		NATSURL:          natstestserver.Start(t),
		OrdersSubject:    DefaultOrdersSubject,
		OrdersDurable:    DefaultOrdersDurable,
		PageStreams:      publishedPageStreams(),
		ProxyURL:         proxy,
		FetchConcurrency: 2,
		RunPageBudget:    DefaultRunPageBudget,
		FrontierCap:      DefaultFrontierCap,
		MaxBodyBytes:     DefaultMaxBodyBytes,
		FetchDeadline:    time.Second,
		OpsAddr:          "127.0.0.1:0",
	}

	publishOrder(t, cfg.NATSURL)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := RunService(
		ctx,
		cfg,
		prometheus.New(),
	); err != nil &&
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunService: %v", err)
	}
}

func TestRunServiceFailsOnEmptyExtractor(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	cfg := ServiceConfig{
		NATSURL: natstestserver.Start(t), OrdersSubject: DefaultOrdersSubject,
		OrdersDurable: DefaultOrdersDurable,
		PageStreams:   publishedPageStreams(),
		ProxyURL:      proxy, FetchConcurrency: 2,
		MaxBodyBytes:  DefaultMaxBodyBytes,
		FetchDeadline: time.Second, OpsAddr: "127.0.0.1:0",
		ContentTypes: []string{"application/unregistered"},
	}
	if err := RunService(context.Background(), cfg, prometheus.New()); err == nil {
		t.Fatal("empty active extractor set should fail startup")
	}
}

func TestRunServiceRejectsBadNATSURL(t *testing.T) {
	cfg := ServiceConfig{NATSURL: "nats://127.0.0.1:1", FetchConcurrency: 2, OpsAddr: "127.0.0.1:0"}
	if err := RunService(context.Background(), cfg, prometheus.New()); err == nil {
		t.Fatal("unreachable nats should fail")
	}
}

func publishOrder(t *testing.T, natsURL string) {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natsURL)
	ctx := context.Background()
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{DefaultOrdersSubject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatal(err)
	}
	payload, err := yacycrawlcontract.MarshalCrawlOrder(yacycrawlcontract.CrawlOrder{
		OrderID: "o1",
		Profile: yacycrawlcontract.CrawlProfile{
			Scope: yacycrawlcontract.ScopeWide, URLMustMatch: yacycrawlcontract.MatchAll,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
		SeedURLs: []string{"http://origin.example/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(ctx, DefaultOrdersSubject, payload); err != nil {
		t.Fatal(err)
	}
}
