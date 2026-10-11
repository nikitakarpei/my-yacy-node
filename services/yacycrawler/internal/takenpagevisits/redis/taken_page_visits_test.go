package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/takenpagevisits/redis"
)

const (
	orderID   = "o1"
	taker     = "YACY_CRAWL_FRONTIER/1"
	retention = time.Hour
)

func takenPageVisitsOn(server *miniredis.Miniredis) *redis.TakenPageVisits {
	return redis.New(goredis.NewClient(&goredis.Options{Addr: server.Addr()}), retention)
}

func takenPageVisits(t *testing.T) *redis.TakenPageVisits {
	t.Helper()
	return takenPageVisitsOn(miniredis.RunT(t))
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func takePageVisit(
	t *testing.T,
	visits *redis.TakenPageVisits,
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

func TestAPageVisitIsTakableAgainOnceItsRetentionPasses(t *testing.T) {
	server := miniredis.RunT(t)
	visits := takenPageVisitsOn(server)
	url := pageURL(t)
	takePageVisit(t, visits, url, taker)

	server.FastForward(retention)

	if !takePageVisit(t, visits, url, "another taker") {
		t.Fatal("a page visit past its retention should be takable again")
	}
}
