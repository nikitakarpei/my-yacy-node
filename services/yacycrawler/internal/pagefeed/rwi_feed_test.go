package pagefeed_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagefeed"
)

func TestRWIFeedPublishesTheIndexBuiltFromTheText(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationKindRWI,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.rwi", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	feed := pagefeed.NewRWIFeed(js, "yacy.crawl.page.rwi")
	publication, err := feed.Derive(samplePage(), []byte("the quick brown fox"))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	if err := feed.Publish(ctx, publication); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(
		t,
		js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationKindRWI),
	)
	chunk, err := yacycrawlcontract.UnmarshalPageRWIChunk(msg)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	metadata, ok := chunk.(yacycrawlcontract.PageRWIMetadataChunk)
	if !ok {
		t.Fatalf("first chunk = %T, want PageRWIMetadataChunk", chunk)
	}
	if metadata.CanonicalURL != samplePage().CanonicalURL {
		t.Fatalf("canonical url = %q", metadata.CanonicalURL)
	}
	if len(metadata.Metadata) != 1 {
		t.Fatalf("metadata rows = %d, want 1", len(metadata.Metadata))
	}
	if metadata.Metadata[0].Title != samplePage().Title {
		t.Fatalf("metadata not carried as built: %+v", metadata.Metadata[0])
	}
}

func TestRWIFeedChunksBoundPostings(t *testing.T) {
	words := make([]string, 2001)
	for i := range words {
		words[i] = fmt.Sprintf("w%d", i)
	}
	feed := pagefeed.NewRWIFeed(nil, "yacy.crawl.page.rwi")
	publication, err := feed.Derive(samplePage(), []byte(strings.Join(words, " ")))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	messages := publication.Messages()
	if len(messages) != 4 {
		t.Fatalf("messages = %d, want 4: metadata plus 1000, 1000 and 1 postings", len(messages))
	}
	for i, message := range messages {
		if _, err := yacycrawlcontract.UnmarshalPageRWIChunk(message); err != nil {
			t.Fatalf("unmarshal chunk %d: %v", i, err)
		}
	}
}

func TestRWIFeedDeclaresRepresentationAndContentFormat(t *testing.T) {
	feed := pagefeed.NewRWIFeed(nil, "yacy.crawl.page.rwi")
	if feed.Representation() != yacycrawlcontract.PageRepresentationKindRWI {
		t.Fatalf("representation = %q", feed.Representation())
	}
	if feed.ContentFormat() != crawlcapability.PageContentFormatDocumentHTML {
		t.Fatalf("content format = %q", feed.ContentFormat())
	}
}
