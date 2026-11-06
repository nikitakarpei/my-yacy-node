package pagescrape_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapeandroid/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const runWait = 10 * time.Second

func environmentFor(natsURL string) string {
	return "SCRAPE_NATS_URL=" + natsURL + "\n" +
		"SCRAPE_PROXY_URL=http://127.0.0.1:1\n" +
		"SCRAPE_PAGE_OFFER_MAX_BYTES=8MB\n" +
		"PAGESCRAPE_OPS_ADDR=127.0.0.1:0\n"
}

func TestInstanceFromRefusesTextThatIsNotAnEnvironment(t *testing.T) {
	if _, err := pagescrape.InstanceFrom("SCRAPE_NATS_URL"); err == nil {
		t.Fatal("want an error for a line without =")
	}
}

func TestInstanceFromRefusesAnInvalidLogLevel(t *testing.T) {
	if _, err := pagescrape.InstanceFrom(
		environmentFor("nats://127.0.0.1:1") + "LOG_LEVEL=loud\n",
	); err == nil {
		t.Fatal("want an error for an invalid log level")
	}
}

func TestInstanceFromRefusesAnEnvironmentWithoutTheBroker(t *testing.T) {
	if _, err := pagescrape.InstanceFrom("SCRAPE_PROXY_URL=http://127.0.0.1:1\n"); err == nil {
		t.Fatal("want an error when the environment names no broker")
	}
}

func TestInstanceRunsTheServiceUntilItIsStopped(t *testing.T) {
	natsURL := natstestserver.Start(t)
	instance, err := pagescrape.InstanceFrom(environmentFor(natsURL))
	if err != nil {
		t.Fatalf("InstanceFrom: %v", err)
	}
	runDone := make(chan error, 1)
	go func() { runDone <- instance.Run() }()

	awaitStream(t, natsURL, pagescrapecontract.ScrapePageOffersStreamName)
	instance.Stop()

	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("Run: %v", err)
		}
	case <-time.After(runWait):
		t.Fatal("Run did not return after Stop")
	}
}

func TestInstanceRunReportsAnUnreachableBroker(t *testing.T) {
	instance, err := pagescrape.InstanceFrom(environmentFor("nats://127.0.0.1:1"))
	if err != nil {
		t.Fatalf("InstanceFrom: %v", err)
	}
	if err := instance.Run(); err == nil {
		t.Fatal("want an error when the broker is unreachable")
	}
}

func awaitStream(t *testing.T, natsURL string, streamName string) {
	t.Helper()
	broker := natstestserver.ConnectJetStream(t, natsURL)
	ctx, cancel := context.WithTimeout(t.Context(), runWait)
	defer cancel()
	for {
		if _, err := broker.Stream(ctx, streamName); err == nil {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("stream %s did not open", streamName)
		case <-time.After(20 * time.Millisecond):
		}
	}
}
