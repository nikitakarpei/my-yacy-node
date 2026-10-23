package visitcrawlassembly_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	visitcrawl "github.com/nikitakarpei/yacy-rwi-node/visitcrawl/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/visitcrawl/internal/visitintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestRunServiceRedirectsAndPlacesOrder(t *testing.T) {
	crawlNATSURL := natstestserver.Start(t)
	cfg := visitcrawl.ServiceConfig{
		CrawlNATSURL:       crawlNATSURL,
		CrawlOrdersSubject: visitcrawl.DefaultCrawlOrdersSubject,
		OrderTimeout:       visitcrawl.DefaultOrderTimeout,
		MaxInFlight:        visitcrawl.DefaultMaxInFlight,
		MaxBodyBytes:       visitcrawl.DefaultMaxBodyBytes,
		LinkSecret:         "shared-secret",
		CrawlProfile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxPagesPerHost: visitcrawl.DefaultCrawlMaxPagesPerHost,
		},
	}
	listeners := visitcrawl.Listeners{Intake: listenerFor(t)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serviceErr := make(chan error, 1)
	registry := prometheus.NewRegistry()
	go func() {
		serviceErr <- visitcrawl.RunService(ctx, cfg, listeners, registry)
	}()

	consumer := ordersConsumer(t, ctx, crawlNATSURL)

	client := &http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	visitReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		signedVisitURL(listeners.Intake.Addr().String(), "https://example.org/a", cfg.LinkSecret),
		nil,
	)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := client.Do(visitReq)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("status = %d, want 302", resp.StatusCode)
	}
	if got := resp.Header.Get("Location"); got != "https://example.org/a" {
		t.Fatalf("location = %q", got)
	}

	if endpoints := servedEndpointsOf(t, registry); len(endpoints) != 1 ||
		endpoints[0] != visitintake.PathVisit {
		t.Errorf("served endpoints = %v, want [%s]", endpoints, visitintake.PathVisit)
	}

	order := nextPlacedOrder(t, consumer)
	if len(order.SeedURLs) != 1 ||
		order.SeedURLs[0] != canonicalurltest.CanonicalURLOf(t, "https://example.org/a") {
		t.Fatalf("order seeds = %v", order.SeedURLs)
	}

	cancel()
	if err := <-serviceErr; err != nil {
		t.Fatalf("RunService: %v", err)
	}
}

func TestRunServiceRejectsBadCrawlNATSURL(t *testing.T) {
	cfg := visitcrawl.ServiceConfig{CrawlNATSURL: "nats://127.0.0.1:1"}
	listeners := visitcrawl.Listeners{Intake: listenerFor(t)}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	registry := prometheus.NewRegistry()
	err := visitcrawl.RunService(ctx, cfg, listeners, registry)
	if err == nil {
		t.Fatal("unreachable nats should fail")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected connect failure, got deadline exceeded")
	}
}

func TestListenersForListensOnTheIntakeAddress(t *testing.T) {
	listeners, err := visitcrawl.ListenersFor(
		t.Context(), visitcrawl.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Intake.Close() })
	if listeners.Intake.Addr().String() == "127.0.0.1:0" {
		t.Errorf("intake listener = %s, want a bound port", listeners.Intake.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := visitcrawl.ListenersFor(
		t.Context(), visitcrawl.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable intake address")
	}
}

func signedVisitURL(listenAddr, visitedPage, secret string) string {
	seconds := strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(seconds + "\n" + visitedPage))
	return fmt.Sprintf("http://%s/visit?url=%s&expires=%s&signature=%s",
		listenAddr, url.QueryEscape(visitedPage), seconds, hex.EncodeToString(seal.Sum(nil)))
}

func servedEndpointsOf(t *testing.T, registry *prometheus.Registry) []string {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	var endpoints []string
	for _, family := range families {
		if family.GetName() != "visitcrawl_http_requests_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "endpoint" {
					endpoints = append(endpoints, label.GetValue())
				}
			}
		}
	}
	return endpoints
}

func ordersConsumer(t *testing.T, ctx context.Context, crawlNATSURL string) jetstream.Consumer {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, crawlNATSURL)
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      yacycrawlcontract.OrdersStreamName,
		Subjects:  []string{visitcrawl.DefaultCrawlOrdersSubject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		t.Fatal(err)
	}
	consumer, err := js.CreateOrUpdateConsumer(ctx, yacycrawlcontract.OrdersStreamName,
		jetstream.ConsumerConfig{
			AckPolicy:     jetstream.AckExplicitPolicy,
			FilterSubject: visitcrawl.DefaultCrawlOrdersSubject,
		})
	if err != nil {
		t.Fatal(err)
	}
	return consumer
}

func nextPlacedOrder(t *testing.T, consumer jetstream.Consumer) yacycrawlcontract.CrawlOrder {
	t.Helper()
	msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("fetch placed order: %v", err)
	}
	order, err := yacycrawlcontract.UnmarshalCrawlOrder(msg.Data())
	if err != nil {
		t.Fatalf("decode order: %v", err)
	}
	return order
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
