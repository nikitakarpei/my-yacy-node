package pagepublication_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagepublication"
)

func TestMarkdownPublicationPublishes(t *testing.T) {
	js := startJetStream(t)
	ctx := context.Background()
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationMarkdown,
		yacycrawlcontract.CrawledPageStreamSpec{Subject: "yacy.crawl.page.markdown", MaxMsgs: 10},
	); err != nil {
		t.Fatal(err)
	}
	publication := pagepublication.NewMarkdownPublication(js, "yacy.crawl.page.markdown")
	representation := crawlcapability.MarkdownRepresentation{
		PageReference: sampleReference(),
		Markdown:      []byte("# hi"),
	}
	if err := publication.Publish(ctx, representation); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg := consumeOne(
		t,
		js,
		yacycrawlcontract.CrawledPageStreamName(yacycrawlcontract.PageRepresentationMarkdown),
	)
	page, err := yacycrawlcontract.UnmarshalPageContentRepresentation(msg)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(page.Body) != "# hi" {
		t.Fatalf("body = %q", page.Body)
	}
}
