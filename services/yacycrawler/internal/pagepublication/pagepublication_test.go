package pagepublication_test

import (
	"context"
	"errors"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagepublication"
)

func startJetStream(t *testing.T) jetstream.JetStream {
	t.Helper()
	srv, err := natsserver.NewServer(&natsserver.Options{
		Port: -1, JetStream: true, StoreDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("nats not ready")
	}
	t.Cleanup(srv.Shutdown)
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	return js
}

type textDerivation struct{}

func (textDerivation) Format() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatText
}

func (textDerivation) SourceFormats() []crawlcapability.PageContentFormat {
	return []crawlcapability.PageContentFormat{crawlcapability.PageContentFormatText}
}

func (textDerivation) Derive(
	body []byte,
	_ crawlcapability.PageContentFormat,
) ([]byte, error) {
	return body, nil
}

func samplePage() crawlcapability.CrawledPage {
	return crawlcapability.CrawledPage{
		CanonicalURL:   "http://example.com/a",
		Title:          "Hi",
		Body:           []byte("the quick brown fox"),
		Format:         crawlcapability.PageContentFormatText,
		Language:       "en",
		CrawledAt:      time.Unix(1_700_000_000, 0),
		LocalLinkCount: 1,
	}
}

func TestPageRWIOutputPublishes(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationRWI,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.rwi", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	output := pagepublication.NewPageRWIOutput(js, "yacy.crawl.page.rwi", textDerivation{})
	if output.Name() != "rwi" {
		t.Fatalf("name = %q", output.Name())
	}
	if err := output.Publish(ctx, samplePage()); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(
		t,
		js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationRWI),
	)
	chunk, err := yacycrawlcontract.UnmarshalPageRWIChunk(msg)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	metadata, ok := chunk.(yacycrawlcontract.PageRWIMetadataChunk)
	if !ok {
		t.Fatalf("first chunk = %T, want PageRWIMetadataChunk", chunk)
	}
	if metadata.CanonicalURL != "http://example.com/a" {
		t.Fatalf("canonical url = %q", metadata.CanonicalURL)
	}
	if len(metadata.Metadata) != 1 {
		t.Fatalf("metadata rows = %d, want 1", len(metadata.Metadata))
	}
}

func TestPageContentOutputPublishes(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationText,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.text", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	output := pagepublication.NewPageContentOutput(js, "yacy.crawl.page.text", textDerivation{})
	if output.Name() != "text" {
		t.Fatalf("name = %q", output.Name())
	}
	if err := output.Publish(ctx, samplePage()); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(
		t,
		js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationText),
	)
	page, err := yacycrawlcontract.UnmarshalPageContentRepresentation(msg)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if page.Title != "Hi" {
		t.Fatalf("title = %q", page.Title)
	}
}

func TestPublishFullStreamIsRetryable(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationText,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.text", MaxMsgs: 1},
	); err != nil {
		t.Fatal(err)
	}
	output := pagepublication.NewPageContentOutput(js, "yacy.crawl.page.text", textDerivation{})
	if err := output.Publish(ctx, samplePage()); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	err := output.Publish(ctx, samplePage())
	var retryable crawlcapability.TransientPublicationError
	if err == nil || !errors.As(err, &retryable) {
		t.Fatalf("full stream should yield TransientPublicationError, got %v", err)
	}
}

func consumeOne(t *testing.T, js jetstream.JetStream, stream string) []byte {
	t.Helper()
	consumer, err := js.CreateOrUpdateConsumer(context.Background(), stream,
		jetstream.ConsumerConfig{AckPolicy: jetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	_ = msg.Ack()
	return msg.Data()
}

func TestOutputsAcceptOnlyTheirDerivationsSourceFormats(t *testing.T) {
	rwi := pagepublication.NewPageRWIOutput(nil, "yacy.crawl.page.rwi", textDerivation{})
	content := pagepublication.NewPageContentOutput(nil, "yacy.crawl.page.text", textDerivation{})

	for _, output := range []crawlcapability.PagePublication{rwi, content} {
		if !output.Accepts(crawlcapability.PageContentFormatText) {
			t.Fatalf("%s refuses its derivation's source format", output.Name())
		}
		if output.Accepts(crawlcapability.PageContentFormatHTML) {
			t.Fatalf("%s accepts a format its derivation does not", output.Name())
		}
	}
}
