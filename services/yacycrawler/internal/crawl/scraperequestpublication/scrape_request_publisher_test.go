package scraperequestpublication_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/scraperequestpublication"
)

type countingObserver struct {
	publications int
}

func (o *countingObserver) ScrapeRequestPublished() {
	o.publications++
}

type recordingScrapeRequests struct {
	urls     []string
	failWith error
}

func (r *recordingScrapeRequests) Publish(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) error {
	if r.failWith != nil {
		return r.failWith
	}
	r.urls = append(r.urls, canonicalURL.String())
	return nil
}

func TestPublishSendsTheCanonicalFormOfTheFinalURL(t *testing.T) {
	observer := &countingObserver{}
	scrapeRequests := &recordingScrapeRequests{}

	err := scraperequestpublication.NewPublisher(observer, scrapeRequests).
		Publish(context.Background(), "HTTP://Host:80/a#section")
	if err != nil {
		t.Fatalf("publish err = %v", err)
	}
	if len(scrapeRequests.urls) != 1 || scrapeRequests.urls[0] != "http://host/a" {
		t.Fatalf("published urls = %v", scrapeRequests.urls)
	}
	if observer.publications != 1 {
		t.Fatalf("observed publications = %d", observer.publications)
	}
}

func TestPublishSkipsAURLItCannotCanonicalize(t *testing.T) {
	observer := &countingObserver{}
	scrapeRequests := &recordingScrapeRequests{}

	err := scraperequestpublication.NewPublisher(observer, scrapeRequests).
		Publish(context.Background(), "ftp://host/a")
	if err != nil {
		t.Fatalf("publish err = %v", err)
	}
	if len(scrapeRequests.urls) != 0 {
		t.Fatalf("published urls = %v", scrapeRequests.urls)
	}
	if observer.publications != 0 {
		t.Fatalf("observed publications = %d", observer.publications)
	}
}

func TestPublishReportsAFailedPublication(t *testing.T) {
	observer := &countingObserver{}
	failure := errors.New("stream down")
	scrapeRequests := &recordingScrapeRequests{failWith: failure}

	err := scraperequestpublication.NewPublisher(observer, scrapeRequests).
		Publish(context.Background(), "http://host/a")
	if !errors.Is(err, failure) {
		t.Fatalf("publish err = %v", err)
	}
	if observer.publications != 0 {
		t.Fatalf("observed publications = %d", observer.publications)
	}
}
