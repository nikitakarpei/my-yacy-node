//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	indexBaseName  = "yacy_text"
	fanOutPrefix   = indexBaseName + "_v1"
	crawledPageMax = 64
)

var crawledPageSubject = yacycrawlcontract.CrawledPageSubject(
	yacycrawlcontract.PageRepresentationKindText,
)

func publishCrawledCorpus(t *testing.T, ctx context.Context, natsURL string) {
	t.Helper()
	nc, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	if err := yacycrawlcontract.EnsureCrawledPageStream(
		ctx,
		js,
		yacycrawlcontract.PageRepresentationKindText,
		yacycrawlcontract.CrawledPageStreamSpec{
			Subject: crawledPageSubject,
			MaxMsgs: crawledPageMax,
		},
	); err != nil {
		t.Fatalf("ensure crawled page stream: %v", err)
	}
	for _, page := range crawledPages() {
		publishCrawledPage(t, ctx, js, page)
	}
}

func publishCrawledPage(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	page yacycrawlcontract.PageTextRepresentation,
) {
	t.Helper()
	data, err := yacycrawlcontract.MarshalPageTextRepresentation(page)
	if err != nil {
		t.Fatalf("marshal crawled page: %v", err)
	}
	if _, err := js.Publish(ctx, crawledPageSubject, data); err != nil {
		t.Fatalf("publish crawled page: %v", err)
	}
}

func crawledPages() []yacycrawlcontract.PageTextRepresentation {
	return []yacycrawlcontract.PageTextRepresentation{
		{
			PageReference: yacycrawlcontract.PageReference{
				CanonicalURL: englishURL,
				Title:        englishTitle,
				CrawledAt:    time.Now().UTC(),
				Language:     englishLanguage,
			},
			Text: []byte(englishContent),
		},
		{
			PageReference: yacycrawlcontract.PageReference{
				CanonicalURL: germanURL,
				Title:        germanTitle,
				CrawledAt:    time.Now().UTC(),
				Language:     germanLanguage,
			},
			Text: []byte(germanContent),
		},
	}
}

const (
	englishLanguage   = "en"
	germanLanguage    = "de"
	englishTitle      = "Riverside Wildflower Guide"
	englishURL        = "https://example.invalid/wildflower-guide"
	englishContent    = "A field guide to wildflowers found along riverside trails."
	englishSearchTerm = "wildflower"
	englishStemmed    = "trail"
	germanTitle       = "Wildblumen am Uferweg"
	germanURL         = "https://example.invalid/wildblumen-uferweg"
	germanContent     = "Ein Feldfuehrer zu Wildblumen an den Uferwegen."
	germanSearchTerm  = "wildblumen"
)
