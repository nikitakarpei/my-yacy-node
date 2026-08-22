package main_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	yacycrawler "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/cmd/yacycrawler"
)

func TestRunServiceProcessesOrderThenStops(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	crawlNATSURL := natstestserver.Start(t)
	cfg := yacycrawler.ServiceConfig{
		CrawlNATSURL:         crawlNATSURL,
		ScrapeRequestNATSURL: crawlNATSURL,
		OrdersSubject:        yacycrawler.DefaultOrdersSubject,
		OrdersDurable:        yacycrawler.DefaultOrdersDurable,
		ProxyURL:             proxy,
		FetchConcurrency:     2,
		RunPageBudget:        yacycrawler.DefaultRunPageBudget,
		FrontierCap:          yacycrawler.DefaultFrontierCap,
		MaxBodyBytes:         yacycrawler.DefaultMaxBodyBytes,
		FetchDeadline:        time.Second,
		OpsAddr:              "127.0.0.1:0",
	}

	publishOrder(t, cfg.CrawlNATSURL)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	registry := prometheus.NewRegistry()
	if err := yacycrawler.RunService(ctx, cfg, registry); err != nil &&
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("yacycrawler.RunService: %v", err)
	}
}

func TestRunServiceRejectsBadCrawlNATSURL(t *testing.T) {
	cfg := yacycrawler.ServiceConfig{
		CrawlNATSURL:         "nats://127.0.0.1:1",
		ScrapeRequestNATSURL: "nats://127.0.0.1:1",
		FetchConcurrency:     2,
		OpsAddr:              "127.0.0.1:0",
	}
	registry := prometheus.NewRegistry()
	if err := yacycrawler.RunService(context.Background(), cfg, registry); err == nil {
		t.Fatal("unreachable nats should fail")
	}
}

func publishOrder(t *testing.T, crawlNATSURL string) {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, crawlNATSURL)
	ctx := context.Background()
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{yacycrawler.DefaultOrdersSubject},
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
		SeedURLs: []canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://origin.example/"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(ctx, yacycrawler.DefaultOrdersSubject, payload); err != nil {
		t.Fatal(err)
	}
}
