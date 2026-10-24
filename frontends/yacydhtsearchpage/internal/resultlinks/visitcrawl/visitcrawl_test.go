package visitcrawl_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/resultlinks/visitcrawl"
)

const secret = "shared-secret"

type fixedClock struct {
	now time.Time
}

func (clock fixedClock) Now() time.Time {
	return clock.now
}

func linkAt(t *testing.T, moment time.Time, address string) url.Values {
	t.Helper()
	link := visitcrawl.New(secret, fixedClock{now: moment}).LinkFor(address)
	path, query, found := strings.Cut(link, "?")
	if !found || path != "/visit" {
		t.Fatalf("link = %q, want /visit with a query", link)
	}
	fields, err := url.ParseQuery(query)
	if err != nil {
		t.Fatalf("parse %q: %v", query, err)
	}
	return fields
}

func TestLinkForCarriesTheSignatureVisitcrawlAccepts(t *testing.T) {
	fields := linkAt(t, time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC), "https://yacy.net/?a=1&b=2")

	if fields.Get("url") != "https://yacy.net/?a=1&b=2" {
		t.Errorf("url = %q, want the address", fields.Get("url"))
	}
	seal := hmac.New(sha256.New, []byte(secret))
	seal.Write([]byte(fields.Get("expires") + "\n" + fields.Get("url")))
	if fields.Get("signature") != hex.EncodeToString(seal.Sum(nil)) {
		t.Errorf("signature = %q, want the HMAC of expiry and address", fields.Get("signature"))
	}
}

func TestLinkForExpiresAtTheEndOfTheNextDay(t *testing.T) {
	fields := linkAt(t, time.Date(2026, 10, 24, 9, 30, 0, 0, time.UTC), "https://yacy.net/")

	want := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC).Unix()
	if fields.Get("expires") != strconv.FormatInt(want, 10) {
		t.Errorf("expires = %q, want %d", fields.Get("expires"), want)
	}
}

func TestLinkForStaysTheSameForTheWholeDay(t *testing.T) {
	morning := visitcrawl.New(
		secret,
		fixedClock{now: time.Date(2026, 10, 24, 0, 1, 0, 0, time.UTC)},
	)
	evening := visitcrawl.New(
		secret,
		fixedClock{now: time.Date(2026, 10, 24, 23, 59, 0, 0, time.UTC)},
	)
	nextDay := visitcrawl.New(
		secret,
		fixedClock{now: time.Date(2026, 10, 25, 0, 1, 0, 0, time.UTC)},
	)

	if morning.LinkFor("https://yacy.net/") != evening.LinkFor("https://yacy.net/") {
		t.Error("links of the same day differ")
	}
	if morning.LinkFor("https://yacy.net/") == nextDay.LinkFor("https://yacy.net/") {
		t.Error("links of different days are the same")
	}
}

func TestLinkForLeavesAnAddressThatIsNotAWebAddress(t *testing.T) {
	links := visitcrawl.New(secret, fixedClock{now: time.Now()})

	if link := links.LinkFor("javascript:alert(1)"); link != "javascript:alert(1)" {
		t.Errorf("LinkFor = %q, want the address unchanged", link)
	}
}
