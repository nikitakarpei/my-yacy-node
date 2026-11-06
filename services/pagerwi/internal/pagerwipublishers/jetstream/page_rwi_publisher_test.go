package jetstream_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	pagerwipublishers "github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwipublishers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
)

const (
	pageAddress = "https://example.org/a"
	readWait    = 5 * time.Second
)

type recordingPublicationObserver struct {
	published          []canonicalurl.CanonicalURL
	encodingFailures   int
	publishingFailures int
}

func (o *recordingPublicationObserver) PageRWIPublished(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	o.published = append(o.published, pageURL)
}

func (o *recordingPublicationObserver) PageRWIEncodingFailed(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
	o.encodingFailures++
}

func (o *recordingPublicationObserver) PageRWIPublishingFailed(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
	o.publishingFailures++
}

func TestPublishedPageRWIReachesTheNodeConsumer(t *testing.T) {
	ctx := context.Background()
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	consumer := pageRWIConsumerOn(t, ctx, broker)
	observer := &recordingPublicationObserver{}

	published := pagerwipublishers.NewPageRWIPublisher(broker, observer).Publish(
		ctx, pageRWIOf(t),
	)

	if !published {
		t.Fatal("publish = false, want the stream to take the rwi")
	}
	message, err := consumer.Next(jetstream.FetchMaxWait(readWait))
	if err != nil {
		t.Fatalf("read the rwi: %v", err)
	}
	read, err := pagerwicontract.UnmarshalPageRWI(message.Data())
	if err != nil {
		t.Fatalf("unmarshal the rwi: %v", err)
	}
	if read.Metadata.Address != pageAddress {
		t.Errorf("read address = %q, want %q", read.Metadata.Address, pageAddress)
	}
	if len(observer.published) != 1 || observer.published[0].String() != pageAddress {
		t.Errorf("published = %v, want the page url once", observer.published)
	}
}

func TestPageRWINoStreamTakesIsReportedAsNotPublished(t *testing.T) {
	broker := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	observer := &recordingPublicationObserver{}

	published := pagerwipublishers.NewPageRWIPublisher(broker, observer).Publish(
		context.Background(), pageRWIOf(t),
	)

	if published {
		t.Fatal("publish = true without a stream, want false")
	}
	if observer.publishingFailures != 1 || len(observer.published) != 0 {
		t.Errorf("observer = %+v, want one publishing failure", observer)
	}
}

func pageRWIOf(t *testing.T) pagerwicontract.PageRWI {
	t.Helper()

	return pagerwicontract.PageRWI{
		PageURL: canonicalurltest.CanonicalURLOf(t, pageAddress),
		Metadata: yacymodel.URLMetadata{
			Hash:    urlhashtest.URLHashOf(t, pageAddress),
			Address: pageAddress,
		},
	}
}

func pageRWIConsumerOn(
	t *testing.T,
	ctx context.Context,
	broker jetstream.JetStream,
) jetstream.Consumer {
	t.Helper()

	stream, err := broker.CreateStream(ctx, jetstream.StreamConfig{
		Name:     pagerwicontract.PageRWIsStreamName,
		Subjects: []string{pagerwicontract.PageRWISubject},
	})
	if err != nil {
		t.Fatalf("create the rwis stream: %v", err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "nodes",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create a node consumer: %v", err)
	}
	return consumer
}
