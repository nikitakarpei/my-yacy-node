package yacycrawlcontract_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestARecrawlGraceOutsideItsBoundsCannotBeBuilt(t *testing.T) {
	for _, length := range []time.Duration{
		-time.Second, yacycrawlcontract.MaxRecrawlGrace.Length() + time.Second,
	} {
		if _, err := yacycrawlcontract.RecrawlGraceOf(length); err == nil {
			t.Fatalf("a recrawl grace of %s was built", length)
		}
	}
}

func TestARecrawlGraceCoversTheTimeBeforeItsLength(t *testing.T) {
	recrawlGrace, err := yacycrawlcontract.RecrawlGraceOf(time.Hour)
	if err != nil {
		t.Fatalf("recrawl grace: %v", err)
	}
	if !recrawlGrace.Covers(59*time.Minute) || recrawlGrace.Covers(time.Hour) {
		t.Fatal("want a 1h grace to cover 59m and not 1h")
	}
}

func TestUnmarshalCrawlOrderReadsARecrawlGraceInNanoseconds(t *testing.T) {
	order, err := yacycrawlcontract.UnmarshalCrawlOrder([]byte(
		`{"OrderID":"o1","Profile":{"RecrawlGrace":3600000000000},"SeedURLs":[]}`,
	))
	if err != nil || order.Profile.RecrawlGrace.Length() != time.Hour {
		t.Fatalf("order = %+v (%v), want a 1h recrawl grace", order, err)
	}
}

func TestUnmarshalCrawlOrderRejectsARecrawlGraceOutsideItsBounds(t *testing.T) {
	for _, body := range []string{
		`{"OrderID":"o1","Profile":{"RecrawlGrace":-1},"SeedURLs":[]}`,
		`{"OrderID":"o1","Profile":{"RecrawlGrace":2592000000000001},"SeedURLs":[]}`,
		`{"OrderID":"o1","Profile":{"RecrawlGrace":"1h"},"SeedURLs":[]}`,
	} {
		if _, err := yacycrawlcontract.UnmarshalCrawlOrder([]byte(body)); err == nil {
			t.Fatalf("UnmarshalCrawlOrder(%s) returned nil, want the grace refused", body)
		}
	}
}
