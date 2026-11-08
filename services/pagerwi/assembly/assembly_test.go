package pagerwiassembly_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pagerwi "github.com/nikitakarpei/yacy-rwi-node/pagerwi/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/jetstreamconnect"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	brokerWait   = 10 * time.Second
	shutdownWait = 10 * time.Second
	pageURL      = "https://example.org/page"
	nodeDurable  = "assemblytest"
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

func TestRunServicePublishesThePageRWIOfAnOfferedPage(t *testing.T) {
	natsURL := natstestserver.Start(t)
	broker := natstestserver.ConnectJetStream(t, natsURL)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	createPageOffersStream(t, ctx, broker)

	health := &servicehealth.Health{}
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagerwi.RunService(
			ctx, serviceConfig(natsURL), pagerwi.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, jetstreamconnect.BrokerAccess{}, pullintake.EveryItemAdmitted{}, prometheus.NewRegistry(), health,
		)
	}()
	pageRWIs := consumerOnceOpen(t, ctx, broker, pagerwicontract.PageRWIsStreamName, nodeDurable)
	awaitConsumer(
		t,
		ctx,
		broker,
		pagescrapecontract.ScrapePageOffersStreamName,
		pagerwi.DefaultPageOfferDurable,
	)
	offerPage(t, ctx, broker, "<html lang=\"en\"><body><p>alpha beta</p></body></html>")

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
	admission := heldAdmission{asked: make(chan struct{}, 1), released: make(chan struct{})}
	runDone := make(chan error, 1)
	go func() {
		runDone <- pagerwi.RunService(
			ctx, serviceConfig(natsURL), pagerwi.Listeners{}, (&net.Dialer{}).DialContext,
			jetstreamconnect.BrokerAccess{}, jetstreamconnect.BrokerAccess{}, admission,
			prometheus.NewRegistry(), &servicehealth.Health{},
		)
	}()
	pageRWIs := consumerOnceOpen(t, ctx, broker, pagerwicontract.PageRWIsStreamName, nodeDurable)
	admission.awaitAsked(t)
	offerPage(t, ctx, broker, "<html lang=\"en\"><body><p>alpha beta</p></body></html>")

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
	close(admission.released)
	if _, err := pageRWIs.Next(jetstream.FetchMaxWait(brokerWait)); err != nil {
		t.Fatalf("await rwi after admission: %v", err)
	}
	cancel()
	if err := <-runDone; err != nil {
		t.Errorf("run: %v", err)
	}
}

type heldAdmission struct {
	asked    chan struct{}
	released chan struct{}
}

func (a heldAdmission) AwaitAdmission(ctx context.Context) error {
	select {
	case a.asked <- struct{}{}:
	default:
	}
	select {
	case <-a.released:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("await admission: %w", ctx.Err())
	}
}

func (a heldAdmission) awaitAsked(t *testing.T) {
	t.Helper()
	select {
	case <-a.asked:
	case <-time.After(brokerWait):
		t.Fatal("the service never asked the intake admission")
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
	pageRWIs := consumerOnceOpen(t, ctx, broker, pagerwicontract.PageRWIsStreamName, nodeDurable)
	offerPage(t, ctx, broker, "<html lang=\"en\"><body><p>alpha beta</p></body></html>")

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

func consumerOnceOpen(
	t *testing.T,
	ctx context.Context,
	broker jetstream.JetStream,
	streamName string,
	durable string,
) jetstream.Consumer {
	t.Helper()
	deadline := time.Now().Add(brokerWait)
	for {
		consumer, err := broker.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
			Durable:   durable,
			AckPolicy: jetstream.AckExplicitPolicy,
		})
		if err == nil {
			return consumer
		}
		if time.Now().After(deadline) {
			t.Fatalf("open the %s consumer on %s: %v", durable, streamName, err)
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

func offerPage(t *testing.T, ctx context.Context, broker jetstream.JetStream, body string) {
	t.Helper()
	data, err := pagescrapecontract.MarshalOfferedPage(pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, pageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, pageURL),
		ContentType: "text/html",
		Body:        []byte(body),
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
