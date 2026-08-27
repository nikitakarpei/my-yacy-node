package jetstream_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitclaims/jetstream"
)

const orderID = "o1"

func visitClaims(t *testing.T) *jetstream.Claims {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	if err := jetstream.Ensure(ctx, js, jetstream.BucketSpec{}); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}
	bucket, err := js.KeyValue(ctx, jetstream.BucketName)
	if err != nil {
		t.Fatalf("open bucket: %v", err)
	}
	return jetstream.New(bucket, jetstream.Config{
		MaxDeferralsPerURL: 2,
		MaxAttemptsPerURL:  2,
	})
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func claim(t *testing.T, claims *jetstream.Claims, url canonicalurl.CanonicalURL) bool {
	t.Helper()
	claimed, err := claims.Claim(context.Background(), orderID, url)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	return claimed
}

func TestOnlyTheFirstClaimOfAURLSucceeds(t *testing.T) {
	claims := visitClaims(t)
	url := pageURL(t)

	if !claim(t, claims, url) {
		t.Fatal("the first claim of a url should succeed")
	}
	if claim(t, claims, url) {
		t.Fatal("the second claim of the same url should fail")
	}
}

func TestTheSameURLIsClaimableAgainUnderAnotherOrder(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()
	url := pageURL(t)

	if !claim(t, claims, url) {
		t.Fatal("the first order should claim the url")
	}
	claimed, err := claims.Claim(ctx, "o2", url)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if !claimed {
		t.Fatal("another order should claim the same url")
	}
}

func TestDeferralsRunOutAtTheirCeiling(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()
	url := pageURL(t)
	claim(t, claims, url)

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
	claim(t, claims, url)

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
	claim(t, claims, url)

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

func TestAHostSpendsNoMorePagesThanItsProfileAllows(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()

	for page := range 2 {
		spent, err := claims.SpendHostPage(ctx, orderID, "host", 2)
		if err != nil {
			t.Fatalf("spend host page: %v", err)
		}
		if !spent {
			t.Fatalf("page %d should be within the host allowance", page+1)
		}
	}
	spent, err := claims.SpendHostPage(ctx, orderID, "host", 2)
	if err != nil {
		t.Fatalf("spend host page: %v", err)
	}
	if spent {
		t.Fatal("a page beyond the host allowance should be refused")
	}
}

func TestAnUnlimitedHostAllowanceSpendsNothing(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()

	for range 3 {
		spent, err := claims.SpendHostPage(
			ctx, orderID, "host", yacycrawlcontract.UnlimitedPagesPerHost,
		)
		if err != nil {
			t.Fatalf("spend host page: %v", err)
		}
		if !spent {
			t.Fatal("an unlimited host allowance should never refuse a page")
		}
	}
}

func TestDeferringAURLNoWorkerClaimedFails(t *testing.T) {
	if _, err := visitClaims(t).Defer(context.Background(), orderID, pageURL(t)); err == nil {
		t.Fatal("a url with no claim should not defer")
	}
}
