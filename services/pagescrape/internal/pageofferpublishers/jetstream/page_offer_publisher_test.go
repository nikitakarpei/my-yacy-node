package jetstream_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pageofferpublishers "github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/pageofferpublishers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const pageURL = "https://example.org/a"

func offerStream(t *testing.T, ctx context.Context) (jetstream.JetStream, jetstream.Consumer) {
	t.Helper()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	stream, err := broker.CreateStream(ctx, jetstream.StreamConfig{
		Name: pagescrapecontract.ScrapePageOffersStreamName,
		Subjects: []string{
			pagescrapecontract.OfferedPageSubject,
			pagescrapecontract.ScrapeFailureSubject,
		},
	})
	if err != nil {
		t.Fatalf("create the page offer stream: %v", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "corpustest",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create a page offer consumer: %v", err)
	}
	return broker, consumer
}

func nextOffer(t *testing.T, consumer jetstream.Consumer) jetstream.Msg {
	t.Helper()
	message, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("take the next offer: %v", err)
	}
	return message
}

func TestOfferedPageReachesTheCorpora(t *testing.T) {
	ctx := context.Background()
	broker, consumer := offerStream(t, ctx)
	page := pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, pageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, pageURL),
		ContentType: "text/html",
		Body:        []byte("page"),
	}

	if err := pageofferpublishers.NewPageOfferPublisher(broker).
		OfferPage(ctx, page); err != nil {
		t.Fatalf("offer the page: %v", err)
	}

	message := nextOffer(t, consumer)
	if message.Subject() != pagescrapecontract.OfferedPageSubject {
		t.Errorf("page offered on %q, want %q",
			message.Subject(), pagescrapecontract.OfferedPageSubject)
	}
	offered, err := pagescrapecontract.UnmarshalOfferedPage(message.Data())
	if err != nil {
		t.Fatalf("unmarshal the offered page: %v", err)
	}
	if offered.PageURL != page.PageURL {
		t.Errorf("offered the page %s, want %s", offered.PageURL, page.PageURL)
	}
}

func TestScrapeFailureReachesTheCorpora(t *testing.T) {
	ctx := context.Background()
	broker, consumer := offerStream(t, ctx)
	failure := pagescrapecontract.ScrapeFailure{
		PageURL:  canonicalurltest.CanonicalURLOf(t, pageURL),
		FetchURL: canonicalurltest.CanonicalURLOf(t, pageURL),
		Reason:   pagescrapecontract.AccessRefused,
	}

	if err := pageofferpublishers.NewPageOfferPublisher(broker).
		ReportScrapeFailure(ctx, failure); err != nil {
		t.Fatalf("report the scrape failure: %v", err)
	}

	message := nextOffer(t, consumer)
	if message.Subject() != pagescrapecontract.ScrapeFailureSubject {
		t.Errorf("failure reported on %q, want %q",
			message.Subject(), pagescrapecontract.ScrapeFailureSubject)
	}
	reported, err := pagescrapecontract.UnmarshalScrapeFailure(message.Data())
	if err != nil {
		t.Fatalf("unmarshal the scrape failure: %v", err)
	}
	if reported != failure {
		t.Errorf("reported %#v, want %#v", reported, failure)
	}
}

func TestOfferWithoutAStreamFails(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	publisher := pageofferpublishers.NewPageOfferPublisher(broker)

	if err := publisher.OfferPage(ctx, pagescrapecontract.OfferedPage{}); err == nil {
		t.Error("want an error offering a page no stream takes")
	}
	if err := publisher.ReportScrapeFailure(
		ctx, pagescrapecontract.ScrapeFailure{},
	); err == nil {
		t.Error("want an error reporting a failure no stream takes")
	}
}
