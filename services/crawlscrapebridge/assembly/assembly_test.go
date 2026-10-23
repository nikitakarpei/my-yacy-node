package crawlscrapebridgeassembly_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	crawlscrapebridge "github.com/nikitakarpei/yacy-rwi-node/crawlscrapebridge/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	crawledPageURL   = "https://example.org/page"
	relayedDeadline  = 5 * time.Second
	relayedPollPause = 50 * time.Millisecond
	shutdownDeadline = 10 * time.Second
)

func serviceConfig(natsURL string) crawlscrapebridge.ServiceConfig {
	return crawlscrapebridge.ServiceConfig{
		CrawlNATSURL:                natsURL,
		ScrapeRequestNATSURL:        natsURL,
		CrawledPageDurable:          crawlscrapebridge.DefaultCrawledPageDurable,
		CrawledPageRelayConcurrency: crawlscrapebridge.DefaultCrawledPageRelayConcurrency,
	}
}

func TestRunServiceAsksToScrapeEveryIndexableCrawledPage(t *testing.T) {
	natsURL := natstestserver.Start(t)
	broker := natstestserver.ConnectJetStream(t, natsURL)
	crawledPages := createStream(t, broker, yacycrawlcontract.CrawledPagesStreamName,
		yacycrawlcontract.EveryCrawledPageSubject)
	scrapeRequests := createStream(t, broker, pagescrapecontract.ScrapeRequestsStreamName,
		pagescrapecontract.ScrapeRequestSubject)
	publishCrawledPage(t, broker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- crawlscrapebridge.RunService(
			ctx, serviceConfig(natsURL), crawlscrapebridge.Listeners{}, prometheus.NewRegistry(),
		)
	}()

	request := awaitScrapeRequest(t, scrapeRequests)
	if want := canonicalurltest.CanonicalURLOf(t, crawledPageURL); request.PageURL != want {
		t.Errorf("scrape request page = %s, want %s", request.PageURL, want)
	}
	awaitAcknowledged(t, crawledPages)

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunServiceStopsCleanlyWhileTheCrawledPagesAreUnavailable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() {
		runDone <- crawlscrapebridge.RunService(
			ctx,
			serviceConfig(natstestserver.Start(t)),
			crawlscrapebridge.Listeners{},
			prometheus.NewRegistry(),
		)
	}()

	cancel()
	awaitShutdown(t, runDone)
}

func TestRunServiceFailsWhenTheScrapeRequestNATSIsAway(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t))
	cfg.ScrapeRequestNATSURL = "nats://127.0.0.1:1"

	if err := crawlscrapebridge.RunService(
		context.Background(), cfg, crawlscrapebridge.Listeners{}, prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("RunService ran while the scrape request nats is away, want an error")
	}
}

func TestListenersForOpensNoListener(t *testing.T) {
	listeners, err := crawlscrapebridge.ListenersFor(t.Context(), crawlscrapebridge.ServiceConfig{})
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	if listeners != (crawlscrapebridge.Listeners{}) {
		t.Errorf("listeners = %+v, want none", listeners)
	}
}

func createStream(
	t *testing.T,
	broker jetstream.JetStream,
	name string,
	subject string,
) jetstream.Stream {
	t.Helper()
	stream, err := broker.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     name,
		Subjects: []string{subject},
	})
	if err != nil {
		t.Fatalf("create the %s stream: %v", name, err)
	}
	return stream
}

func publishCrawledPage(t *testing.T, broker jetstream.JetStream) {
	t.Helper()
	data, err := yacycrawlcontract.MarshalCrawledPage(yacycrawlcontract.CrawledPage{
		PageURL: canonicalurltest.CanonicalURLOf(t, crawledPageURL),
	})
	if err != nil {
		t.Fatalf("marshal crawled page: %v", err)
	}
	if _, err := broker.Publish(
		context.Background(), yacycrawlcontract.IndexablePageSubject, data,
	); err != nil {
		t.Fatalf("publish crawled page: %v", err)
	}
}

func awaitScrapeRequest(
	t *testing.T,
	scrapeRequests jetstream.Stream,
) pagescrapecontract.ScrapeRequest {
	t.Helper()
	deadline := time.Now().Add(relayedDeadline)
	for time.Now().Before(deadline) {
		message, err := scrapeRequests.GetLastMsgForSubject(
			context.Background(), pagescrapecontract.ScrapeRequestSubject,
		)
		if err == nil {
			request, err := pagescrapecontract.UnmarshalScrapeRequest(message.Data)
			if err != nil {
				t.Fatalf("unmarshal the scrape request: %v", err)
			}
			return request
		}
		time.Sleep(relayedPollPause)
	}
	t.Fatal("no scrape request arrived")
	return pagescrapecontract.ScrapeRequest{}
}

func awaitAcknowledged(t *testing.T, crawledPages jetstream.Stream) {
	t.Helper()
	deadline := time.Now().Add(relayedDeadline)
	for time.Now().Before(deadline) {
		consumer, err := crawledPages.Consumer(
			context.Background(), crawlscrapebridge.DefaultCrawledPageDurable,
		)
		if err == nil && acknowledgedOne(consumer) {
			return
		}
		time.Sleep(relayedPollPause)
	}
	t.Fatal("the crawled page was never acknowledged")
}

func acknowledgedOne(consumer jetstream.Consumer) bool {
	info, err := consumer.Info(context.Background())
	return err == nil && info.AckFloor.Stream == 1
}

func awaitShutdown(t *testing.T, runDone <-chan error) {
	t.Helper()
	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("RunService: %v", err)
		}
	case <-time.After(shutdownDeadline):
		t.Fatal("RunService did not stop after cancel")
	}
}
