package unattested_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/offerattesters/unattested"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

var clockTime = time.Date(2026, time.November, 10, 9, 0, 0, 0, time.UTC)

func TestAPageIsFetchedAtTheClockTime(t *testing.T) {
	fetchedAt := unattested.New(fixedClock{}).FetchTimeOf(
		context.Background(), redirectfollowingfetch.LandedFetch{},
	)

	if !fetchedAt.Equal(clockTime) {
		t.Errorf("fetched at %v, want %v", fetchedAt, clockTime)
	}
}

func TestEveryOfferGoesWithoutAttestationAndIsDecided(t *testing.T) {
	bundle, decided := unattested.New(fixedClock{}).Attest(
		context.Background(),
		pagescrapecontract.PageOfferMessage{Body: []byte("page")},
		redirectfollowingfetch.LandedFetch{},
	)

	if !decided || len(bundle) != 0 {
		t.Errorf("bundle = %s, decided = %v", bundle, decided)
	}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return clockTime
}
