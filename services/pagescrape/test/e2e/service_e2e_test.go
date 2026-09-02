//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

func TestRequestedPageIsOfferedToTheCorpora(t *testing.T) {
	ctx := context.Background()
	js, offers := startScrapeStack(t, ctx)

	publishScrapeRequest(t, ctx, js, originPageURL)

	message := nextMessage(t, offers, offerWait)
	if message == nil {
		t.Fatalf("no page offered within %s", offerWait)
	}
	if message.Subject() != pagescrapecontract.OfferedPageSubject {
		t.Fatalf("page offered on %q, want %q, body %q",
			message.Subject(), pagescrapecontract.OfferedPageSubject, message.Data())
	}
	offered, err := pagescrapecontract.UnmarshalOfferedPage(message.Data())
	if err != nil {
		t.Fatalf("unmarshal offered page: %v", err)
	}
	if offered.PageURL != canonicalurltest.CanonicalURLOf(t, originPageURL) {
		t.Errorf("offered page URL = %s, want %s", offered.PageURL, originPageURL)
	}
	if !strings.Contains(string(offered.Body), originBody) {
		t.Errorf("offered body = %q, want it to contain %q", offered.Body, originBody)
	}
}

func TestPageTheOriginDoesNotServeIsReportedAsAScrapeFailureOnce(t *testing.T) {
	ctx := context.Background()
	js, offers := startScrapeStack(t, ctx)

	publishScrapeRequest(t, ctx, js, missingPageURL)

	message := nextMessage(t, offers, offerWait)
	if message == nil {
		t.Fatalf("no scrape failure reported within %s", offerWait)
	}
	if message.Subject() != pagescrapecontract.ScrapeFailureSubject {
		t.Fatalf("scrape reported on %q, want %q, body %q",
			message.Subject(), pagescrapecontract.ScrapeFailureSubject, message.Data())
	}
	failure, err := pagescrapecontract.UnmarshalScrapeFailure(message.Data())
	if err != nil {
		t.Fatalf("unmarshal scrape failure: %v", err)
	}
	if failure.PageURL != canonicalurltest.CanonicalURLOf(t, missingPageURL) {
		t.Errorf("failed page URL = %s, want %s", failure.PageURL, missingPageURL)
	}
	if failure.Reason != pagescrapecontract.AccessRefused {
		t.Errorf("failure reason = %s, want %s", failure.Reason, pagescrapecontract.AccessRefused)
	}
	if repeated := nextMessage(t, offers, silenceWait); repeated != nil {
		t.Errorf("the request was scraped again: %s %q", repeated.Subject(), repeated.Data())
	}
}
