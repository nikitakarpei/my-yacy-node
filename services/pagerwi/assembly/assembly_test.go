package pagerwiassembly_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pagerwi "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	brokerWait   = 10 * time.Second
	shutdownWait = 10 * time.Second
	pageURL      = "https://example.org/page"
	pageBody     = "<html lang=\"en\"><body><p>alpha beta</p></body></html>"
	nodeDurable  = "assemblytest"
	everyInbox   = "_INBOX.>"

	pageOfferPassword = "page-offer-password"
	pageRWIPassword   = "page-rwi-password"
)

func serviceConfig(natsURL string) pagerwi.ServiceConfig {
	return pagerwi.ServiceConfig{
		PageOfferNATSURL:           natsURL,
		PageOfferDurable:           pagerwi.DefaultPageOfferDurable,
		PageOfferIntakeConcurrency: pagerwi.DefaultPageOfferIntakeConcurrency,
		PageRWINATSURL:             natsURL,
		PageRWIMaxBytes:            8 << 20,
		PageRWIPartitionExponent:   pagerwi.DefaultPageRWIPartitionExponent,
		ProvisionsStreams:          true,
	}
}

func TestRunServicePublishesThePageRWIOfAnOfferedPageWithTheAccessOfEachBroker(t *testing.T) {
	pageOfferURL := natstestserver.StartWithPassword(t, pageOfferPassword)
	pageRWIURL := natstestserver.StartWithPassword(t, pageRWIPassword)
	pageOfferBroker := natstestserver.ConnectJetStream(
		t, pageOfferURL, nats.UserInfo("", pageOfferPassword),
	)
	pageRWIBroker := natstestserver.ConnectJetStream(
		t,
		pageRWIURL,
		nats.UserInfo("", pageRWIPassword),
	)
	cfg := serviceConfig(pageOfferURL)
	cfg.PageRWINATSURL = pageRWIURL
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createPageOffersStream(t, ctx, pageOfferBroker)

	health := &servicehealth.Health{}
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagerwi.RunService(
			ctx, cfg, pagerwi.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{Password: pageOfferPassword},
			jetstreamconnect.BrokerAccess{Password: pageRWIPassword},
			pullintake.EveryItemAdmitted{}, prometheus.NewRegistry(), health,
		)
	}()
	pageRWIs := pageRWIConsumerOnceOpen(t, ctx, pageRWIBroker)
	awaitConsumer(
		t,
		ctx,
		pageOfferBroker,
		pagescrapecontract.ScrapePageOffersStreamName,
		pagerwi.DefaultPageOfferDurable,
	)
	offerPage(t, ctx, pageOfferBroker)

	message, err := pageRWIs.Next(jetstream.FetchMaxWait(brokerWait))
	if err != nil {
		t.Fatalf("await rwi: %v", err)
	}
	pageRWI, err := pagerwicontract.UnmarshalPageRWI(message.Data())
	if err != nil {
		t.Fatalf("unmarshal rwi: %v", err)
	}
	if pageRWI.Metadata.Address != pageURL {
		t.Errorf("rwi address = %q, want %q", pageRWI.Metadata.Address, pageURL)
	}
	if !hasPostingOf(pageRWI, "alpha") || !pageRWI.Metadata.PostingsRoot.Present() {
		t.Errorf("rwi = %+v, want a posting of alpha under a postings root", pageRWI)
	}
	if !health.Started() {
		t.Error("health not started while the service publishes RWIs")
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

func TestRunServiceTakesNoPageOfferUntilTheIntakeAdmissionAdmits(t *testing.T) {
	natsURL := natstestserver.Start(t)
	broker := natstestserver.ConnectJetStream(t, natsURL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createPageOffersStream(t, ctx, broker)
	release := make(chan struct{})
	admission := pullintaketest.AdmissionHeldUntil(release)
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagerwi.RunService(
			ctx, serviceConfig(natsURL), pagerwi.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, jetstreamconnect.BrokerAccess{}, admission,
			prometheus.NewRegistry(), &servicehealth.Health{},
		)
	}()
	pageRWIs := pageRWIConsumerOnceOpen(t, ctx, broker)
	admission.AwaitAsked(t, brokerWait)
	offerPage(t, ctx, broker)

	pageOffers, err := broker.Consumer(
		ctx, pagescrapecontract.ScrapePageOffersStreamName, pagerwi.DefaultPageOfferDurable,
	)
	if err != nil {
		t.Fatalf("open page offer consumer: %v", err)
	}
	info, err := pageOffers.Info(ctx)
	if err != nil {
		t.Fatalf("page offer consumer info: %v", err)
	}
	if info.NumPending != 1 || info.NumAckPending != 0 {
		t.Errorf("pending = %d, taken = %d, want the page offer left in the stream",
			info.NumPending, info.NumAckPending)
	}
	close(release)
	if _, err := pageRWIs.Next(jetstream.FetchMaxWait(brokerWait)); err != nil {
		t.Fatalf("await rwi after admission: %v", err)
	}
	cancel()
	if err := <-runDone; err != nil {
		t.Errorf("run: %v", err)
	}
}

func TestRunServiceReturnsWhenThePageRWIBrokerIsUnreachable(t *testing.T) {
	cfg := serviceConfig("nats://127.0.0.1:1")
	health := &servicehealth.Health{}
	if err := pagerwi.RunService(
		context.Background(),
		cfg,
		pagerwi.Listeners{},
		(&net.Dialer{}).DialContext,
		jetstreamconnect.BrokerAccess{},
		jetstreamconnect.BrokerAccess{},
		pullintake.EveryItemAdmitted{},
		prometheus.NewRegistry(),
		health,
	); err == nil {
		t.Fatal("expected an error when the broker is unreachable")
	}
	if health.Started() {
		t.Error("health started although the broker is unreachable")
	}
}

func TestRunServiceReturnsOnAPartitionExponentOutOfRange(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t))
	cfg.PageRWIPartitionExponent = 63
	if err := pagerwi.RunService(
		context.Background(),
		cfg,
		pagerwi.Listeners{},
		(&net.Dialer{}).DialContext,
		jetstreamconnect.BrokerAccess{},
		jetstreamconnect.BrokerAccess{},
		pullintake.EveryItemAdmitted{},
		prometheus.NewRegistry(),
		&servicehealth.Health{},
	); err == nil {
		t.Fatal("expected an error for a partition exponent out of range")
	}
}

func TestRunServiceWithoutProvisioningBuildsRWIsOnTheStreamsThatExistAndKeepsThem(t *testing.T) {
	natsURL := natstestserver.Start(t)
	broker := natstestserver.ConnectJetStream(t, natsURL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createPageOffersStream(t, ctx, broker)
	const provisionedMaxBytes, provisionedPending = 1 << 20, 2
	if _, err := broker.CreateStream(ctx, jetstream.StreamConfig{
		Name:      pagerwicontract.PageRWIsStreamName,
		Subjects:  []string{pagerwicontract.PageRWISubject},
		Retention: jetstream.InterestPolicy,
		MaxBytes:  provisionedMaxBytes,
	}); err != nil {
		t.Fatalf("create page RWIs stream: %v", err)
	}
	if _, err := broker.CreateConsumer(
		ctx,
		pagescrapecontract.ScrapePageOffersStreamName,
		jetstream.ConsumerConfig{
			Durable:       pagerwi.DefaultPageOfferDurable,
			FilterSubject: pagescrapecontract.OfferedPageSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
			MaxAckPending: provisionedPending,
		},
	); err != nil {
		t.Fatalf("create page offer consumer: %v", err)
	}
	cfg := serviceConfig(natsURL)
	cfg.ProvisionsStreams = false

	runDone := make(chan error, 1)
	go func() {
		runDone <- pagerwi.RunService(
			ctx, cfg, pagerwi.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
			prometheus.NewRegistry(), &servicehealth.Health{},
		)
	}()
	pageRWIs := pageRWIConsumerOnceOpen(t, ctx, broker)
	offerPage(t, ctx, broker)

	if _, err := pageRWIs.Next(jetstream.FetchMaxWait(brokerWait)); err != nil {
		t.Fatalf("await rwi: %v", err)
	}
	stream, err := broker.Stream(ctx, pagerwicontract.PageRWIsStreamName)
	if err != nil {
		t.Fatalf("open page RWIs stream: %v", err)
	}
	if maxBytes := stream.CachedInfo().Config.MaxBytes; maxBytes != provisionedMaxBytes {
		t.Errorf("page RWIs max bytes = %d, want the provisioned %d", maxBytes, provisionedMaxBytes)
	}
	consumer, err := broker.Consumer(
		ctx,
		pagescrapecontract.ScrapePageOffersStreamName,
		pagerwi.DefaultPageOfferDurable,
	)
	if err != nil {
		t.Fatalf("open page offer consumer: %v", err)
	}
	if pending := consumer.CachedInfo().Config.MaxAckPending; pending != provisionedPending {
		t.Errorf("page offers pending = %d, want the provisioned %d", pending, provisionedPending)
	}
	cancel()
	<-runDone
}

func TestRunServiceAsAUserWithOnlyItsBrokerPermissionsPublishesThePageRWI(t *testing.T) {
	permissions := serviceConfig("").BrokerPermissions()
	urls := natstestserver.StartWithLimitedUser(t, natstestserver.AllowedSubjects{
		Publish: permissions.Publish, Subscribe: append(permissions.Subscribe, everyInbox),
	})
	unlimitedURL, limitedURL := urls.Unlimited, urls.Limited
	broker := natstestserver.ConnectJetStream(t, unlimitedURL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createPageOffersStream(t, ctx, broker)
	if _, err := broker.CreateStream(ctx, jetstream.StreamConfig{
		Name:      pagerwicontract.PageRWIsStreamName,
		Subjects:  []string{pagerwicontract.PageRWISubject},
		Retention: jetstream.InterestPolicy,
	}); err != nil {
		t.Fatalf("create page RWIs stream: %v", err)
	}
	if _, err := broker.CreateConsumer(ctx, pagescrapecontract.ScrapePageOffersStreamName,
		jetstream.ConsumerConfig{
			Durable:       pagerwi.DefaultPageOfferDurable,
			FilterSubject: pagescrapecontract.OfferedPageSubject,
			AckPolicy:     jetstream.AckExplicitPolicy,
		},
	); err != nil {
		t.Fatalf("create page offer consumer: %v", err)
	}
	pageRWIs := pageRWIConsumerOnceOpen(t, ctx, broker)
	intakeReceipts, err := natstestserver.Connect(t, unlimitedURL).SubscribeSync(
		pagescrapecontract.EveryIntakeReceiptSubject,
	)
	if err != nil {
		t.Fatalf("subscribe to intake receipts: %v", err)
	}
	limitedConfig := serviceConfig(limitedURL)
	limitedConfig.ProvisionsStreams = false
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagerwi.RunService(
			ctx, limitedConfig, pagerwi.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{},
			prometheus.NewRegistry(), &servicehealth.Health{},
		)
	}()
	offerPage(t, ctx, broker)

	if _, err := pageRWIs.Next(jetstream.FetchMaxWait(brokerWait)); err != nil {
		t.Fatalf("await rwi: %v", err)
	}
	if _, err := intakeReceipts.NextMsg(brokerWait); err != nil {
		t.Errorf("await the intake receipt: %v", err)
	}
	cancel()
	if err := <-runDone; err != nil {
		t.Errorf("run: %v", err)
	}
}

func TestRunServiceWithoutProvisioningFailsWithoutThePageRWIsStream(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t))
	cfg.ProvisionsStreams = false

	if err := pagerwi.RunService(
		context.Background(),
		cfg,
		pagerwi.Listeners{},
		(&net.Dialer{}).DialContext,
		jetstreamconnect.BrokerAccess{},
		jetstreamconnect.BrokerAccess{},
		pullintake.EveryItemAdmitted{},
		prometheus.NewRegistry(),
		&servicehealth.Health{},
	); err == nil {
		t.Fatal("RunService = nil, want the missing page RWIs stream refused")
	}
}

func TestListenersForOpensNoListener(t *testing.T) {
	listeners, err := pagerwi.ListenersFor(t.Context(), pagerwi.ServiceConfig{})
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	if listeners != (pagerwi.Listeners{}) {
		t.Errorf("listeners = %+v, want none", listeners)
	}
	if err := listeners.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func createPageOffersStream(t *testing.T, ctx context.Context, broker jetstream.JetStream) {
	t.Helper()
	if _, err := broker.CreateStream(ctx, jetstream.StreamConfig{
		Name:      pagescrapecontract.ScrapePageOffersStreamName,
		Subjects:  []string{pagescrapecontract.OfferedPageSubject},
		Retention: jetstream.InterestPolicy,
	}); err != nil {
		t.Fatalf("create the page offers stream: %v", err)
	}
}

func pageRWIConsumerOnceOpen(
	t *testing.T,
	ctx context.Context,
	broker jetstream.JetStream,
) jetstream.Consumer {
	t.Helper()
	deadline := time.Now().Add(brokerWait)
	for {
		consumer, err := broker.CreateOrUpdateConsumer(
			ctx, pagerwicontract.PageRWIsStreamName, jetstream.ConsumerConfig{
				Durable:   nodeDurable,
				AckPolicy: jetstream.AckExplicitPolicy,
			},
		)
		if err == nil {
			return consumer
		}
		if time.Now().After(deadline) {
			t.Fatalf("open the page RWI consumer: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func awaitConsumer(
	t *testing.T,
	ctx context.Context,
	broker jetstream.JetStream,
	streamName string,
	durable string,
) {
	t.Helper()
	deadline := time.Now().Add(brokerWait)
	for {
		_, err := broker.Consumer(ctx, streamName, durable)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("await the %s consumer on %s: %v", durable, streamName, err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func offerPage(t *testing.T, ctx context.Context, broker jetstream.JetStream) {
	t.Helper()
	data, err := pagescrapecontract.MarshalOfferedPage(pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, pageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, pageURL),
		ContentType: "text/html",
		Body:        []byte(pageBody),
	})
	if err != nil {
		t.Fatalf("marshal offered page: %v", err)
	}
	if _, err := broker.Publish(ctx, pagescrapecontract.OfferedPageSubject, data); err != nil {
		t.Fatalf("offer the page: %v", err)
	}
}

func hasPostingOf(pageRWI pagerwicontract.PageRWI, word string) bool {
	for _, posting := range pageRWI.Postings {
		if posting.WordHash == yacymodel.WordHash(word) {
			return true
		}
	}
	return false
}
