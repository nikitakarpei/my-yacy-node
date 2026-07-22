package jetstream_test

import (
	"context"
	"errors"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/feedpublishers/jetstream"
)

const testRepresentation yacycrawlcontract.PageRepresentationKind = "test"

type fakeFramer struct{}

func (fakeFramer) Representation() yacycrawlcontract.PageRepresentationKind {
	return testRepresentation
}

func (fakeFramer) ContentFormat() contentformatgraph.Format {
	return contentformatgraph.FormatFullText
}

func (fakeFramer) Frame(
	_ pageabsorption.CrawledPage,
	content []byte,
) (pageabsorption.PagePublication, error) {
	return pageabsorption.NewPagePublication(content), nil
}

func startJetStream(t *testing.T) natsjetstream.JetStream {
	t.Helper()
	return natstestserver.ConnectJetStream(t, natstestserver.Start(t))
}

func consumeOne(t *testing.T, js natsjetstream.JetStream, stream string) []byte {
	t.Helper()
	consumer, err := js.CreateOrUpdateConsumer(context.Background(), stream,
		natsjetstream.ConsumerConfig{AckPolicy: natsjetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(natsjetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	_ = msg.Ack()
	return msg.Data()
}

func TestWrapPublishesFramedMessages(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		testRepresentation,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.test", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}

	feed := jetstream.Wrap(fakeFramer{}, "yacy.crawl.page.test", js)
	publication, err := feed.Frame(pageabsorption.CrawledPage{}, []byte("hello"))
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if err := feed.Publish(ctx, publication); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(t, js, yacycrawlcontract.CrawledPageStreamName(testRepresentation))
	if string(msg) != "hello" {
		t.Fatalf("message = %q, want %q", msg, "hello")
	}
}

func TestWrapFullStreamIsRetryable(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		testRepresentation,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.test.full", MaxMsgs: 1},
	); err != nil {
		t.Fatal(err)
	}

	feed := jetstream.Wrap(fakeFramer{}, "yacy.crawl.page.test.full", js)
	publication, err := feed.Frame(pageabsorption.CrawledPage{}, []byte("hello"))
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if err := feed.Publish(ctx, publication); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	var retryable pageabsorption.TransientPublicationError
	if err := feed.Publish(ctx, publication); err == nil || !errors.As(err, &retryable) {
		t.Fatalf("full stream should yield TransientPublicationError, got %v", err)
	}
}
