package jetstream_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/takenpagevisits/jetstream"
)

const (
	orderID = "o1"
	taker   = "yacy.crawl.frontier YACY_CRAWL_FRONTIER/1"
)

func takenPageVisits(t *testing.T) *jetstream.TakenPageVisits {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	if err := jetstream.Ensure(ctx, js, jetstreamrecord.BucketSpec{}); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}
	bucket, err := js.KeyValue(ctx, jetstream.BucketName)
	if err != nil {
		t.Fatalf("open bucket: %v", err)
	}
	return jetstream.New(bucket)
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func takePageVisit(
	t *testing.T,
	visits *jetstream.TakenPageVisits,
	url canonicalurl.CanonicalURL,
	taker string,
) bool {
	t.Helper()
	took, err := visits.TakePageVisit(context.Background(), orderID, url, taker)
	if err != nil {
		t.Fatalf("TakePageVisit: %v", err)
	}
	return took
}

func TestTheFirstTakerTakesThePageVisitAndTheNextDoesNot(t *testing.T) {
	visits := takenPageVisits(t)
	url := pageURL(t)

	if !takePageVisit(t, visits, url, taker) {
		t.Fatal("the first taker should take the page visit")
	}
	if takePageVisit(t, visits, url, "another taker") {
		t.Fatal("a second taker should find the page visit taken")
	}
}

func TestTheTakerOfAPageVisitTakesItAgainOnRedelivery(t *testing.T) {
	visits := takenPageVisits(t)
	url := pageURL(t)
	takePageVisit(t, visits, url, taker)

	if !takePageVisit(t, visits, url, taker) {
		t.Fatal("the taker should keep the page visit it already took")
	}
}

func TestTheSameURLIsTakableAgainUnderAnotherOrder(t *testing.T) {
	visits := takenPageVisits(t)
	url := pageURL(t)
	takePageVisit(t, visits, url, taker)

	took, err := visits.TakePageVisit(context.Background(), "o2", url, taker)
	if err != nil {
		t.Fatalf("TakePageVisit: %v", err)
	}
	if !took {
		t.Fatal("another order should take the same url")
	}
}
