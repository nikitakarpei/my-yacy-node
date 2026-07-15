package pagepublication_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestRWIPublicationPublishes(t *testing.T) {
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
	publication := pagepublication.NewRWIPublication(js, "yacy.crawl.page.rwi")
	representation := crawlcapability.RWIRepresentation{
		PageReference:  sampleReference(),
		TextLength:     19,
		WordCount:      4,
		LocalLinkCount: 1,
		Postings: []yacymodel.RWIPosting{
			{WordHash: yacymodel.WordHash("fox"), Properties: map[string]string{}},
		},
	}
	if err := publication.Publish(ctx, representation); err != nil {
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
	if metadata.CanonicalURL != representation.CanonicalURL {
		t.Fatalf("canonical url = %q", metadata.CanonicalURL)
	}
	if len(metadata.Metadata) != 1 {
		t.Fatalf("metadata rows = %d, want 1", len(metadata.Metadata))
	}
	row := metadata.Metadata[0]
	if _, err := yacymodel.ParseURIMetadataRow(row.String()); err != nil {
		t.Fatalf("metadata row not parseable: %v", err)
	}
	title, err := row.Title(t.Context())
	if err != nil {
		t.Fatalf("Title: %v", err)
	}
	if title != representation.Title {
		t.Fatalf("title = %q, want %q", title, representation.Title)
	}
}

func TestRWIPublicationChunksBoundPostings(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationRWI,
		yacycrawlcontract.CrawledPageStreamSpec{
			Subject: "yacy.crawl.page.rwi.bounded",
			MaxMsgs: 10,
		},
	); err != nil {
		t.Fatal(err)
	}
	postings := make([]yacymodel.RWIPosting, 2001)
	for i := range postings {
		postings[i] = yacymodel.RWIPosting{
			WordHash:   yacymodel.WordHash("w"),
			Properties: map[string]string{},
		}
	}
	publication := pagepublication.NewRWIPublication(js, "yacy.crawl.page.rwi.bounded")
	representation := crawlcapability.RWIRepresentation{
		PageReference: sampleReference(),
		Postings:      postings,
	}
	if err := publication.Publish(ctx, representation); err != nil {
		t.Fatalf("publish: %v", err)
	}

	stream := yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationRWI)
	consumer, err := js.CreateOrUpdateConsumer(ctx, stream,
		jetstream.ConsumerConfig{AckPolicy: jetstream.AckExplicitPolicy})
	if err != nil {
		t.Fatal(err)
	}
	for i, wantChunks := 0, 4; i < wantChunks; i++ {
		msg, err := consumer.Next(jetstream.FetchMaxWait(5 * time.Second))
		if err != nil {
			t.Fatalf("consume chunk %d: %v", i, err)
		}
		_ = msg.Ack()
		if _, err := yacycrawlcontract.UnmarshalPageRWIChunk(msg.Data()); err != nil {
			t.Fatalf("unmarshal chunk %d: %v", i, err)
		}
	}
}
