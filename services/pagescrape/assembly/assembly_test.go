package pagescrapeassembly_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pagescrape "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
)

const (
	offerWait     = 10 * time.Second
	shutdownWait  = 10 * time.Second
	originPageURL = "http://origin.example/"
	everyInbox    = "_INBOX.>"
	receiptRetry  = 100 * time.Millisecond
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

func originAnswering(t *testing.T, status int) *url.URL {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
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
		ProvisionsStreams:       true,
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

	health := &servicehealth.Health{}
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(ctx, cfg, pagescrape.Listeners{}, (&net.Dialer{}).DialContext, prometheus.NewRegistry(), health)
	}()

	offers := pageOfferConsumer(t, ctx, broker)
	publishScrapeRequest(t, ctx, broker)

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
	if !health.Started() {
		t.Error("health not started while the service offers pages")
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
	health := &servicehealth.Health{}
	if err := pagescrape.RunService(
		context.Background(),
		cfg,
		pagescrape.Listeners{},
		(&net.Dialer{}).DialContext,
		prometheus.NewRegistry(),
		health,
	); err == nil {
		t.Fatal("expected an error when the broker is unreachable")
	}
	if health.Started() {
		t.Error("health started although the broker is unreachable")
	}
}

func TestRunServiceWithoutProvisioningScrapesOnTheStreamsThatExistAndKeepsThem(t *testing.T) {
	natsURL := natstestserver.Start(t)
	origin := originServing(t, "<html lang=\"en\"><body>words here</body></html>")
	broker := natstestserver.ConnectJetStream(t, natsURL)
	provisioning := serviceConfig(natsURL, origin)
	provisioning.ScrapeRequestsKept = 7
	runUntilStarted(t, provisioning)

	bound := serviceConfig(natsURL, origin)
	bound.ProvisionsStreams = false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(
			ctx, bound, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			prometheus.NewRegistry(), &servicehealth.Health{},
		)
	}()
	offers := pageOfferConsumer(t, ctx, broker)
	publishScrapeRequest(t, ctx, broker)

	if _, err := offers.Next(jetstream.FetchMaxWait(offerWait)); err != nil {
		t.Fatalf("await page offer: %v", err)
	}
	scrapeRequests, err := broker.Stream(ctx, pagescrapecontract.ScrapeRequestsStreamName)
	if err != nil {
		t.Fatalf("open scrape requests: %v", err)
	}
	if kept := scrapeRequests.CachedInfo().Config.MaxMsgs; kept != 7 {
		t.Errorf("scrape requests kept = %d, want the provisioned 7", kept)
	}
	consumer, err := scrapeRequests.Consumer(ctx, pagescrape.DefaultScrapeRequestDurable)
	if err != nil {
		t.Fatalf("open scrape request consumer: %v", err)
	}
	if pending := consumer.CachedInfo().Config.MaxAckPending; pending != 7 {
		t.Errorf("scrape requests pending = %d, want the provisioned 7", pending)
	}
	cancel()
	<-runDone
}

func TestRunServiceAsAUserWithOnlyItsBrokerPermissionsOffersThePage(t *testing.T) {
	origin := originServing(t, "<html lang=\"en\"><body>words here</body></html>")
	broker := runAsLimitedUser(t, origin)
	offers := pageOfferConsumer(t, t.Context(), broker.jetStream)

	publishScrapeRequest(t, t.Context(), broker.jetStream)

	message, err := offers.Next(jetstream.FetchMaxWait(offerWait))
	if err != nil {
		t.Fatalf("await page offer: %v", err)
	}
	if message.Subject() != pagescrapecontract.OfferedPageSubject {
		t.Errorf("subject = %q, want an offered page", message.Subject())
	}
}

func TestRunServiceAsAUserWithOnlyItsBrokerPermissionsReportsAFailedScrape(t *testing.T) {
	broker := runAsLimitedUser(t, originAnswering(t, http.StatusNotFound))
	offers := pageOfferConsumer(t, t.Context(), broker.jetStream)
	outcomes := subscribed(t, broker.connection, pagescrapecontract.ScrapeOutcomeSubjectsOf(
		canonicalurltest.CanonicalURLOf(t, originPageURL),
	))

	publishScrapeRequest(t, t.Context(), broker.jetStream)

	message, err := offers.Next(jetstream.FetchMaxWait(offerWait))
	if err != nil {
		t.Fatalf("await the failure: %v", err)
	}
	if message.Subject() != pagescrapecontract.ScrapeFailureSubject {
		t.Errorf("subject = %q, want a scrape failure", message.Subject())
	}
	if _, err := outcomes.NextMsg(offerWait); err != nil {
		t.Errorf("await the failed scrape outcome: %v", err)
	}
}

func TestRunServiceAsAUserWithOnlyItsBrokerPermissionsCarriesIntakeReceipts(t *testing.T) {
	broker := runAsLimitedUser(t, originServing(t, "<html></html>"))
	pageURL := canonicalurltest.CanonicalURLOf(t, originPageURL)
	outcomes := subscribed(
		t,
		broker.connection,
		pagescrapecontract.ScrapeOutcomeSubjectsOf(pageURL),
	)

	deadline := time.Now().Add(offerWait)
	for {
		if err := broker.connection.Publish(
			pagescrapecontract.KeptPageSubjectOf(pageURL), nil,
		); err != nil {
			t.Fatalf("publish an intake receipt: %v", err)
		}
		if _, err := outcomes.NextMsg(receiptRetry); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the intake receipt was never carried")
		}
	}
}

type unlimitedBroker struct {
	connection *nats.Conn
	jetStream  jetstream.JetStream
}

func runAsLimitedUser(t *testing.T, origin *url.URL) unlimitedBroker {
	t.Helper()
	permissions := serviceConfig("", origin).BrokerPermissions()
	unlimitedURL, limitedURL := natstestserver.StartWithLimitedUser(
		t, permissions.Publish, append(permissions.Subscribe, everyInbox),
	)
	runUntilStarted(t, serviceConfig(unlimitedURL, origin))
	limited := serviceConfig(limitedURL, origin)
	limited.ProvisionsStreams = false
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(
			ctx, limited, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			prometheus.NewRegistry(), &servicehealth.Health{},
		)
	}()
	t.Cleanup(func() {
		cancel()
		if err := <-runDone; err != nil {
			t.Errorf("run: %v", err)
		}
	})
	connection := natstestserver.Connect(t, unlimitedURL)
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	return unlimitedBroker{connection: connection, jetStream: js}
}

func subscribed(t *testing.T, connection *nats.Conn, subject string) *nats.Subscription {
	t.Helper()
	subscription, err := connection.SubscribeSync(subject)
	if err != nil {
		t.Fatalf("subscribe to %s: %v", subject, err)
	}
	t.Cleanup(func() { _ = subscription.Unsubscribe() })
	return subscription
}

func runUntilStarted(t *testing.T, cfg pagescrape.ServiceConfig) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	health := &servicehealth.Health{}
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(
			ctx, cfg, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			prometheus.NewRegistry(), health,
		)
	}()
	deadline := time.Now().Add(offerWait)
	for !health.Started() {
		if time.Now().After(deadline) {
			t.Fatal("service never started")
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if err := <-runDone; err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunServiceWithoutProvisioningFailsWithoutTheStreams(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t), originServing(t, "<html></html>"))
	cfg.ProvisionsStreams = false
	health := &servicehealth.Health{}

	if err := pagescrape.RunService(
		context.Background(), cfg, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
		prometheus.NewRegistry(), health,
	); err == nil {
		t.Fatal("RunService = nil, want the missing streams refused")
	}
	if health.Started() {
		t.Error("health started without the streams")
	}
}

func TestListenersForOpensNoListener(t *testing.T) {
	listeners, err := pagescrape.ListenersFor(t.Context(), pagescrape.ServiceConfig{})
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	if listeners != (pagescrape.Listeners{}) {
		t.Errorf("listeners = %+v, want none", listeners)
	}
	if err := listeners.Close(); err != nil {
		t.Errorf("Close: %v", err)
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

func publishScrapeRequest(t *testing.T, ctx context.Context, broker jetstream.JetStream) {
	t.Helper()
	data, err := pagescrapecontract.MarshalScrapeRequest(pagescrapecontract.ScrapeRequest{
		PageURL: canonicalurltest.CanonicalURLOf(t, originPageURL),
	})
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	if _, err := broker.Publish(ctx, pagescrapecontract.ScrapeRequestSubject, data); err != nil {
		t.Fatalf("publish scrape request: %v", err)
	}
}
