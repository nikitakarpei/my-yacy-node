package pagescrapeassembly_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pagescrape "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
)

const (
	offerWait     = 10 * time.Second
	shutdownWait  = 10 * time.Second
	originPageURL = "http://origin.example/"
	everyInbox    = "_INBOX.>"

	pullRequestPollPause = 10 * time.Millisecond

	brokerPassword = "broker-password"
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

func TestRunServiceOffersThePageAScrapeRequestNamesWithTheAccessOfTheBroker(t *testing.T) {
	natsURL := natstestserver.StartWithPassword(t, brokerPassword)
	cfg := serviceConfig(
		natsURL,
		originServing(t, "<html lang=\"en\"><body>words here</body></html>"),
	)
	broker := natstestserver.ConnectJetStream(t, natsURL, nats.UserInfo("", brokerPassword))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	health, started := healthReportingStart()
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(ctx, cfg, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{Password: brokerPassword}, pullintake.EveryItemAdmitted{},
			prometheus.NewRegistry(), health)
	}()
	awaitStart(t, started, runDone)

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

func TestRunServiceTakesNoScrapeRequestUntilTheIntakeAdmissionAdmits(t *testing.T) {
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(
		natsURL,
		originServing(t, "<html lang=\"en\"><body>words here</body></html>"),
	)
	broker := natstestserver.ConnectJetStream(t, natsURL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	release := make(chan struct{})
	admission := pullintaketest.AdmissionHeldUntil(release)
	health, started := healthReportingStart()
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(
			ctx, cfg, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, admission, prometheus.NewRegistry(), health,
		)
	}()
	awaitStart(t, started, runDone)
	offers := pageOfferConsumer(t, ctx, broker)
	publishScrapeRequest(t, ctx, broker)
	admission.AwaitAsked(t, offerWait)

	scrapeRequests, err := broker.Consumer(
		ctx, pagescrapecontract.ScrapeRequestsStreamName, pagescrape.DefaultScrapeRequestDurable,
	)
	if err != nil {
		t.Fatalf("open scrape request consumer: %v", err)
	}
	pullintaketest.RequireItemLeftInStream(t, ctx, scrapeRequests, offerWait)
	close(release)
	if _, err := offers.Next(jetstream.FetchMaxWait(offerWait)); err != nil {
		t.Fatalf("await page offer after admission: %v", err)
	}
	cancel()
	if err := <-runDone; err != nil {
		t.Errorf("run: %v", err)
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
		jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
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
	provisionStreams(t, provisioning)

	bound := serviceConfig(natsURL, origin)
	bound.ProvisionsStreams = false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(
			ctx, bound, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
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
	if err := <-runDone; err != nil {
		t.Errorf("run: %v", err)
	}
}

func TestRunServiceAsAUserWithOnlyItsBrokerPermissionsOffersThePage(t *testing.T) {
	origin := originServing(t, "<html lang=\"en\"><body>words here</body></html>")
	unlimitedClient := runAsLimitedUser(t, origin)
	offers := pageOfferConsumer(t, t.Context(), unlimitedClient.jetStream)

	publishScrapeRequest(t, t.Context(), unlimitedClient.jetStream)

	message, err := offers.Next(jetstream.FetchMaxWait(offerWait))
	if err != nil {
		t.Fatalf("await page offer: %v", err)
	}
	if message.Subject() != pagescrapecontract.OfferedPageSubject {
		t.Errorf("subject = %q, want an offered page", message.Subject())
	}
}

func TestRunServiceAsAUserWithOnlyItsBrokerPermissionsReportsAFailedScrape(t *testing.T) {
	unlimitedClient := runAsLimitedUser(t, originAnswering(t, http.StatusNotFound))
	offers := pageOfferConsumer(t, t.Context(), unlimitedClient.jetStream)
	outcomes := subscriptionTo(
		t,
		unlimitedClient.connection,
		pagescrapecontract.ScrapeOutcomeSubjectsOf(
			canonicalurltest.CanonicalURLOf(t, originPageURL),
		),
	)

	publishScrapeRequest(t, t.Context(), unlimitedClient.jetStream)

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
	unlimitedClient := runAsLimitedUser(t, originServing(t, "<html></html>"))
	pageURL := canonicalurltest.CanonicalURLOf(t, originPageURL)
	outcomes := subscriptionTo(
		t,
		unlimitedClient.connection,
		pagescrapecontract.ScrapeOutcomeSubjectsOf(pageURL),
	)

	if err := unlimitedClient.connection.Publish(
		pagescrapecontract.KeptPageSubjectOf(pageURL), nil,
	); err != nil {
		t.Fatalf("publish an intake receipt: %v", err)
	}
	if _, err := outcomes.NextMsg(offerWait); err != nil {
		t.Errorf("await the carried intake receipt: %v", err)
	}
}

type unlimitedClient struct {
	connection *nats.Conn
	jetStream  jetstream.JetStream
}

func runAsLimitedUser(t *testing.T, origin *url.URL) unlimitedClient {
	t.Helper()
	permissions := serviceConfig("", origin).BrokerPermissions()
	urls := natstestserver.StartWithLimitedUser(t, natstestserver.AllowedSubjects{
		Publish: permissions.Publish, Subscribe: append(permissions.Subscribe, everyInbox),
	})
	unlimitedURL, limitedURL := urls.Unlimited, urls.Limited
	provisionStreams(t, serviceConfig(unlimitedURL, origin))
	limitedConfig := serviceConfig(limitedURL, origin)
	limitedConfig.ProvisionsStreams = false
	ctx, cancel := context.WithCancel(context.Background())
	health, started := healthReportingStart()
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(
			ctx, limitedConfig, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
			prometheus.NewRegistry(), health,
		)
	}()
	awaitStart(t, started, runDone)
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
	return unlimitedClient{connection: connection, jetStream: js}
}

func subscriptionTo(t *testing.T, connection *nats.Conn, subject string) *nats.Subscription {
	t.Helper()
	subscription, err := connection.SubscribeSync(subject)
	if err != nil {
		t.Fatalf("subscribe to %s: %v", subject, err)
	}
	return subscription
}

func provisionStreams(t *testing.T, cfg pagescrape.ServiceConfig) {
	t.Helper()
	runUntilStarted(t, cfg)
	awaitPullRequestsDropped(
		t, natstestserver.ConnectJetStream(t, cfg.ScrapeNATSURL), cfg.ScrapeRequestDurable,
	)
}

func runUntilStarted(t *testing.T, cfg pagescrape.ServiceConfig) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	health, started := healthReportingStart()
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagescrape.RunService(
			ctx, cfg, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
			prometheus.NewRegistry(), health,
		)
	}()
	awaitStart(t, started, runDone)
	cancel()
	if err := <-runDone; err != nil {
		t.Fatalf("run: %v", err)
	}
}

func healthReportingStart() (*servicehealth.Health, <-chan struct{}) {
	started := make(chan struct{})
	var startOnce sync.Once
	var health *servicehealth.Health
	health = servicehealth.ReportingTo(func() {
		if health.Started() {
			startOnce.Do(func() { close(started) })
		}
	})
	return health, started
}

func awaitStart(t *testing.T, started <-chan struct{}, runDone <-chan error) {
	t.Helper()
	select {
	case <-started:
	case err := <-runDone:
		t.Fatalf("run ended before the start: %v", err)
	}
}

func awaitPullRequestsDropped(t *testing.T, broker jetstream.JetStream, durable string) {
	t.Helper()
	consumer, err := broker.Consumer(
		t.Context(),
		pagescrapecontract.ScrapeRequestsStreamName,
		durable,
	)
	if err != nil {
		t.Fatalf("open scrape request consumer: %v", err)
	}
	for {
		info, err := consumer.Info(t.Context())
		if err != nil {
			t.Fatalf("scrape request consumer info: %v", err)
		}
		if info.NumWaiting == 0 {
			return
		}
		time.Sleep(pullRequestPollPause)
	}
}

func TestRunServiceWithoutProvisioningFailsWithoutTheStreams(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t), originServing(t, "<html></html>"))
	cfg.ProvisionsStreams = false
	health := &servicehealth.Health{}

	if err := pagescrape.RunService(
		context.Background(), cfg, pagescrape.Listeners{}, (&net.Dialer{}).DialContext,
		jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
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
	consumer, err := broker.CreateOrUpdateConsumer(
		ctx,
		pagescrapecontract.ScrapePageOffersStreamName,
		jetstream.ConsumerConfig{
			Durable:   "assemblytest",
			AckPolicy: jetstream.AckExplicitPolicy,
		},
	)
	if err != nil {
		t.Fatalf("open page offer consumer: %v", err)
	}
	return consumer
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
