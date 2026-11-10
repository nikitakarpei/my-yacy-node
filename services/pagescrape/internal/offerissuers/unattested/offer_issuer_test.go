package unattested_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/offerissuers/unattested"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

var clockTime = time.Date(2026, time.November, 10, 9, 0, 0, 0, time.UTC)

func TestAPageIsOfferedWithoutAttestationAtTheClockTime(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")
	landed := redirectfollowingfetch.LandedFetch{
		URL:     pageURL,
		Outcome: pagefetch.FetchOutcome{Page: pagefetch.FetchedPage{Body: []byte("page")}},
	}

	offer, issued := unattested.New(fixedClock{}).Issue(
		context.Background(),
		pagescrapecontract.ScrapeRequest{PageURL: pageURL, FetchURL: pageURL},
		landed,
	)
	page, err := pagescrapecontract.UnmarshalOfferedPage(offer.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !issued || offer.Attestation.Present() || !page.FetchedAt.Equal(clockTime) ||
		string(page.Body) != "page" {
		t.Errorf("page = %+v, issued = %v", page, issued)
	}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return clockTime
}
