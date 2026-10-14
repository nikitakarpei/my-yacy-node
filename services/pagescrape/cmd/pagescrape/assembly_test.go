package main_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pagescrape "github.com/nikitakarpei/yacy-rwi-node/pagescrape/cmd/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

const (
	offerWait     = 10 * time.Second
	shutdownWait  = 10 * time.Second
	originPageURL = "http://origin.example/"
)

func originServing(t *testing.T, body string) *url.URL {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	proxy, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse origin url: %v", err)
	}
	return proxy
}

func serviceConfig(natsURL string, proxy *url.URL) pagescrape.ServiceConfig {
	return pagescrape.ServiceConfig{
		ScrapeNATSURL:           natsURL,
		ProxyURL:                proxy,
		UserAgent:               pagescrape.DefaultUserAgent,
		MaxBodyBytes:            pagescrape.DefaultScrapeMaxBodyBytes,
		FetchDeadline:           time.Second,
		ScrapeRequestDurable:    pagescrape.DefaultScrapeRequestDurable,
		ScrapeIntakeConcurrency: pagescrape.DefaultScrapeIntakeConcurrency,
		ScrapeRequestsKept:      pagescrape.DefaultScrapeRequestsKept,
		ScrapeDeferralWindow:    pagescrape.DefaultScrapeDeferralWindow,
		PageOfferMaxBytes:       8 << 20,
	}
}

func TestRunServiceOffersThePageAScrapeRequestNames(t *testing.T) {
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(
		natsURL,
		originServing(t, "<html lang=\"en\"><body>words here</body></html>"),
	)
	broker := natstestserver.ConnectJetStream(t, natsURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listeners := pagescrape.Listeners{Ops: listenerFor(t)}
	runDone := make(chan error, 1)
	go func() { runDone <- pagescrape.RunService(ctx, cfg, listeners) }()

	offers := pageOfferConsumer(t, ctx, broker)
	publishScrapeRequest(t, ctx, broker, originPageURL)

	message, err := offers.Next(jetstream.FetchMaxWait(offerWait))
	if err != nil {
		t.Fatalf("await page offer: %v", err)
	}
	offered, err := pagescrapecontract.UnmarshalOfferedPage(message.Data())
	if err != nil {
		t.Fatalf("unmarshal offered page: %v", err)
	}
	if offered.PageURL != canonicalurltest.CanonicalURLOf(t, originPageURL) {
		t.Errorf("offered page = %q", offered.PageURL)
	}

	cancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("run: %v", err)
		}
	case <-time.After(shutdownWait):
		t.Fatal("service did not shut down after cancel")
	}
}

func TestRunServiceReturnsWhenTheBrokerIsUnreachable(t *testing.T) {
	cfg := serviceConfig("nats://127.0.0.1:1", originServing(t, "<html></html>"))
	listeners := pagescrape.Listeners{Ops: listenerFor(t)}

	if err := pagescrape.RunService(context.Background(), cfg, listeners); err == nil {
		t.Fatal("expected an error when the broker is unreachable")
	}
}

func pageOfferConsumer(
	t *testing.T,
	ctx context.Context,
	broker jetstream.JetStream,
) jetstream.Consumer {
	t.Helper()
	deadline := time.Now().Add(offerWait)
	for {
		consumer, err := broker.CreateOrUpdateConsumer(
			ctx,
			pagescrapecontract.ScrapePageOffersStreamName,
			jetstream.ConsumerConfig{
				Durable:   "assemblytest",
				AckPolicy: jetstream.AckExplicitPolicy,
			},
		)
		if err == nil {
			return consumer
		}
		if time.Now().After(deadline) {
			t.Fatalf("open page offer consumer: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func publishScrapeRequest(
	t *testing.T,
	ctx context.Context,
	broker jetstream.JetStream,
	pageURL string,
) {
	t.Helper()
	data, err := pagescrapecontract.MarshalScrapeRequest(pagescrapecontract.ScrapeRequest{
		PageURL: canonicalurltest.CanonicalURLOf(t, pageURL),
	})
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	if _, err := broker.Publish(ctx, pagescrapecontract.ScrapeRequestSubject, data); err != nil {
		t.Fatalf("publish scrape request: %v", err)
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
