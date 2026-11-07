package pageservices_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/connectlink"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeercompanion/pageservices"
)

const (
	runWait = 10 * time.Second
	pageURL = "http://origin.example/"
)

type observedEvents chan string

func (events observedEvents) ServicesConnected()    { events <- "connected" }
func (events observedEvents) BrokerConnectionLost() { events <- "connection lost" }
func (events observedEvents) ServicesStopped()      { events <- "stopped" }
func (events observedEvents) BrokerUnreachable()    { events <- "broker unreachable" }
func (events observedEvents) ServicesFailed()       { events <- "failed" }

func linkTo(natsURL string) string {
	return connectlink.Scheme + "://" + connectlink.Host + "?" + url.Values{
		connectlink.ParameterNATSURL:           {natsURL},
		connectlink.ParameterPartitionExponent: {"4"},
	}.Encode()
}

func captureProxyServing(t *testing.T, body string) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func awaitEvent(t *testing.T, events observedEvents, want string) {
	t.Helper()
	select {
	case event := <-events:
		if event != want {
			t.Fatalf("event = %q, want %q", event, want)
		}
	case <-time.After(runWait):
		t.Fatalf("no %q event", want)
	}
}

func TestInstanceForRefusesTextThatIsNotAConnectLink(t *testing.T) {
	if _, err := pageservices.InstanceFor(
		"https://example.org/",
		"http://127.0.0.1:1",
	); err == nil {
		t.Fatal("want an error for text that is not a connect link")
	}
}

func TestInstanceBuildsTheRWIOfAPageThatTheBrokerAsksFor(t *testing.T) {
	natsURL := natstestserver.Start(t)
	instance, err := pageservices.InstanceFor(
		linkTo(natsURL),
		captureProxyServing(t, "<html lang=\"en\"><body><p>alpha beta</p></body></html>"),
	)
	if err != nil {
		t.Fatalf("InstanceFor: %v", err)
	}
	events := make(observedEvents, 4)
	go instance.Run(events)
	awaitEvent(t, events, "connected")

	broker := natstestserver.ConnectJetStream(t, natsURL)
	pageRWIs, err := broker.CreateOrUpdateConsumer(t.Context(), pagerwicontract.PageRWIsStreamName,
		jetstream.ConsumerConfig{Durable: "node", AckPolicy: jetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatalf("open the RWI consumer: %v", err)
	}
	askForPage(t, broker)
	message, err := pageRWIs.Next(jetstream.FetchMaxWait(runWait))
	if err != nil {
		t.Fatalf("await the RWI: %v", err)
	}
	pageRWI, err := pagerwicontract.UnmarshalPageRWI(message.Data())
	if err != nil {
		t.Fatalf("unmarshal the RWI: %v", err)
	}
	if pageRWI.Metadata.Address != pageURL {
		t.Errorf("RWI address = %q, want %q", pageRWI.Metadata.Address, pageURL)
	}

	instance.Stop()
	awaitEvent(t, events, "stopped")
}

func TestInstanceRunReportsAnUnreachableBroker(t *testing.T) {
	instance, err := pageservices.InstanceFor(linkTo("nats://127.0.0.1:1"), "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("InstanceFor: %v", err)
	}
	events := make(observedEvents, 4)
	instance.Run(events)
	awaitEvent(t, events, "broker unreachable")
}

func TestInstanceRunReportsABrokerThatShutsDown(t *testing.T) {
	broker := natstestserver.StartedServer(t)
	instance, err := pageservices.InstanceFor(linkTo(broker.ClientURL()), "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("InstanceFor: %v", err)
	}
	events := make(observedEvents, 4)
	go instance.Run(events)
	awaitEvent(t, events, "connected")

	broker.Shutdown()
	for {
		select {
		case event := <-events:
			switch event {
			case "connection lost":
				continue
			case "broker unreachable":
				return
			default:
				t.Fatalf("event = %q, want the broker reported unreachable", event)
			}
		case <-time.After(runWait):
			t.Fatal("the run did not end after the broker shut down")
		}
	}
}

func TestInstanceRunReportsAStopBeforeItStarted(t *testing.T) {
	instance, err := pageservices.InstanceFor(linkTo(natstestserver.Start(t)), "http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("InstanceFor: %v", err)
	}
	instance.Stop()
	events := make(observedEvents, 4)
	instance.Run(events)
	awaitEvent(t, events, "stopped")
}

func askForPage(t *testing.T, broker jetstream.JetStream) {
	t.Helper()
	data, err := pagescrapecontract.MarshalScrapeRequest(pagescrapecontract.ScrapeRequest{
		PageURL: canonicalurltest.CanonicalURLOf(t, pageURL),
	})
	if err != nil {
		t.Fatalf("marshal the scrape request: %v", err)
	}
	if _, err := broker.Publish(
		t.Context(),
		pagescrapecontract.ScrapeRequestSubject,
		data,
	); err != nil {
		t.Fatalf("publish the scrape request: %v", err)
	}
}
