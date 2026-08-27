package jetstream_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitclaim"
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

func TestAHostSpendsNoMorePagesThanItsProfileAllows(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()

	for page := range 2 {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/page%d", page))
		claim(t, claims, url, holder)
		spent, err := claims.SpendHostPage(ctx, orderID, url, "host", 2)
		if err != nil {
			t.Fatalf("spend host page: %v", err)
		}
		if !spent {
			t.Fatalf("page %d should be within the host allowance", page+1)
		}
	}
	beyond := canonicalurltest.CanonicalURLOf(t, "http://host/page2")
	claim(t, claims, beyond, holder)
	spent, err := claims.SpendHostPage(ctx, orderID, beyond, "host", 2)
	if err != nil {
		t.Fatalf("spend host page: %v", err)
	}
	if spent {
		t.Fatal("a page beyond the host allowance should be refused")
	}
}

func TestAVisitSpendsOneHostPageHoweverOftenItIsResumed(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()
	url := pageURL(t)
	claim(t, claims, url, holder)

	for resumption := range 3 {
		spent, err := claims.SpendHostPage(ctx, orderID, url, "host", 1)
		if err != nil {
			t.Fatalf("spend host page: %v", err)
		}
		if !spent {
			t.Fatalf("resumption %d should spend the page the visit already holds", resumption)
		}
	}

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	claim(t, claims, other, holder)
	spent, err := claims.SpendHostPage(ctx, orderID, other, "host", 1)
	if err != nil {
		t.Fatalf("spend host page: %v", err)
	}
	if spent {
		t.Fatal("the resumed visit should have spent the whole allowance of the host")
	}
}

func TestAHostThatAllowsNoPagesSpendsNone(t *testing.T) {
	claims := visitClaims(t)
	url := pageURL(t)
	claim(t, claims, url, holder)

	spent, err := claims.SpendHostPage(context.Background(), orderID, url, "host", 0)
	if err != nil {
		t.Fatalf("spend host page: %v", err)
	}
	if spent {
		t.Fatal("a host that allows no pages should refuse the first page")
	}
}

func TestOnlyOneOfTwoConcurrentVisitsSpendsTheLastPageOfAHost(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()
	spends := make(chan bool, 2)

	var spending sync.WaitGroup
	for page := range 2 {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/race%d", page))
		claim(t, claims, url, holder)
		spending.Add(1)
		go func() {
			defer spending.Done()
			spent, err := claims.SpendHostPage(ctx, orderID, url, "host", 1)
			if err != nil {
				t.Errorf("spend host page: %v", err)
				return
			}
			spends <- spent
		}()
	}
	spending.Wait()
	close(spends)

	granted := 0
	for spent := range spends {
		if spent {
			granted++
		}
	}
	if granted != 1 {
		t.Fatalf("%d of two concurrent visits spent a host allowance of one, want 1", granted)
	}
}

func TestAnUnlimitedHostAllowanceSpendsNothing(t *testing.T) {
	claims := visitClaims(t)
	ctx := context.Background()

	for range 3 {
		spent, err := claims.SpendHostPage(
			ctx, orderID, pageURL(t), "host", yacycrawlcontract.UnlimitedPagesPerHost,
		)
		if err != nil {
			t.Fatalf("spend host page: %v", err)
		}
		if !spent {
			t.Fatal("an unlimited host allowance should never refuse a page")
		}
	}
}

func TestDeferringAURLNoHolderClaimedFails(t *testing.T) {
	if _, err := visitClaims(t).Defer(context.Background(), orderID, pageURL(t)); err == nil {
		t.Fatal("a url with no claim should not defer")
	}
}
