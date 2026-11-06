package pagerwi_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwiandroid/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
)

const runWait = 10 * time.Second

func environmentFor(natsURL string) string {
	return "SCRAPE_PAGE_OFFER_NATS_URL=" + natsURL + "\n" +
		"PAGE_RWI_NATS_URL=" + natsURL + "\n" +
		"PAGE_RWI_MAX_BYTES=8MB\n" +
		"PAGERWI_OPS_ADDR=127.0.0.1:0\n"
}

func TestInstanceFromRefusesTextThatIsNotAnEnvironment(t *testing.T) {
	if _, err := pagerwi.InstanceFrom("PAGE_RWI_NATS_URL"); err == nil {
		t.Fatal("want an error for a line without =")
	}
}

func TestInstanceFromRefusesAnInvalidLogLevel(t *testing.T) {
	if _, err := pagerwi.InstanceFrom(
		environmentFor("nats://127.0.0.1:1") + "LOG_LEVEL=loud\n",
	); err == nil {
		t.Fatal("want an error for an invalid log level")
	}
}

func TestInstanceFromRefusesAnEnvironmentWithoutTheBrokers(t *testing.T) {
	if _, err := pagerwi.InstanceFrom("PAGERWI_OPS_ADDR=127.0.0.1:0\n"); err == nil {
		t.Fatal("want an error when the environment names no broker")
	}
}

func TestInstanceRunsTheServiceUntilItIsStopped(t *testing.T) {
	natsURL := natstestserver.Start(t)
	instance, err := pagerwi.InstanceFrom(environmentFor(natsURL))
	if err != nil {
		t.Fatalf("InstanceFrom: %v", err)
	}
	runDone := make(chan error, 1)
	go func() { runDone <- instance.Run() }()

	awaitStream(t, natsURL, pagerwicontract.PageRWIsStreamName)
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
	instance, err := pagerwi.InstanceFrom(environmentFor("nats://127.0.0.1:1"))
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
