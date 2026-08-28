package jetstream_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitclaim"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitclaims/jetstream"
)

const (
	orderID = "o1"
	holder  = "yacy.crawl.frontier YACY_CRAWL_FRONTIER/1"
)

func visitClaims(t *testing.T) *jetstream.Claims {
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
	return jetstream.New(bucket, jetstream.ClaimLimits{
		MaxDeferralsPerURL: 2,
		MaxAttemptsPerURL:  2,
	})
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func claim(
	t *testing.T,
	claims *jetstream.Claims,
	url canonicalurl.CanonicalURL,
	holder string,
) visitclaim.Claim {
	t.Helper()
	claimed, err := claims.Claim(context.Background(), orderID, url, holder)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	return claimed
}

func TestTheFirstHolderTakesTheClaimAndTheNextFindsItHeldElsewhere(t *testing.T) {
	claims := visitClaims(t)
	url := pageURL(t)

	if taken := claim(t, claims, url, holder); taken != visitclaim.Taken {
		t.Fatalf("the first holder got %q, want the claim taken", taken)
	}
	if next := claim(t, claims, url, "another holder"); next != visitclaim.HeldElsewhere {
		t.Fatalf("a second holder got %q, want the claim held elsewhere", next)
	}
}

func TestTheHolderOfAClaimResumesItsOwnVisit(t *testing.T) {
	claims := visitClaims(t)
	url := pageURL(t)
	claim(t, claims, url, holder)

	if resumed := claim(t, claims, url, holder); resumed != visitclaim.Resumed {
		t.Fatalf("the holder got %q, want its own claim resumed", resumed)
	}
}

func TestTheSameURLIsClaimableAgainUnderAnotherOrder(t *testing.T) {
	claims := visitClaims(t)
	url := pageURL(t)
	claim(t, claims, url, holder)

	claimed, err := claims.Claim(context.Background(), "o2", url, holder)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if claimed != visitclaim.Taken {
		t.Fatalf("another order got %q, want the same url taken", claimed)
	}
}

func TestDeferralsRunOutAtTheirCeiling(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()
	url := pageURL(t)
	claim(t, claims, url, holder)

	for attempt := range 2 {
		deferred, err := claims.Defer(ctx, orderID, url)
		if err != nil {
			t.Fatalf("defer: %v", err)
		}
		if !deferred {
			t.Fatalf("deferral %d should be within the ceiling", attempt+1)
		}
	}
	deferred, err := claims.Defer(ctx, orderID, url)
	if err != nil {
		t.Fatalf("defer: %v", err)
	}
	if deferred {
		t.Fatal("a deferral beyond the ceiling should be refused")
	}
}

func TestEachRetryReportsItsAttemptUntilTheCeiling(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()
	url := pageURL(t)
	claim(t, claims, url, holder)

	for want := 1; want <= 2; want++ {
		attempt, retried, err := claims.Retry(ctx, orderID, url)
		if err != nil {
			t.Fatalf("retry: %v", err)
		}
		if !retried || attempt != want {
			t.Fatalf("retry reported attempt %d retried %v, want %d true", attempt, retried, want)
		}
	}
	if _, retried, err := claims.Retry(ctx, orderID, url); err != nil || retried {
		t.Fatalf("a retry beyond the ceiling should be refused, got %v %v", retried, err)
	}
}

func TestDeferralsAndAttemptsKeepSeparateCeilings(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()
	url := pageURL(t)
	claim(t, claims, url, holder)

	for range 2 {
		if _, err := claims.Defer(ctx, orderID, url); err != nil {
			t.Fatalf("defer: %v", err)
		}
	}
	_, retried, err := claims.Retry(ctx, orderID, url)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !retried {
		t.Fatal("spent deferrals should leave the attempts untouched")
	}
}

func TestDeferringAURLNoHolderClaimedFails(t *testing.T) {
	if _, err := visitClaims(t).Defer(context.Background(), orderID, pageURL(t)); err == nil {
		t.Fatal("a url with no claim should not defer")
	}
}
