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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitlimits/jetstream"
)

const (
	orderID = "o1"
	host    = "host"
)

func pageVisitLimits(t *testing.T) *jetstream.PageVisitLimits {
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
	return jetstream.New(bucket, jetstream.MaxPerURL{Deferrals: 2, Attempts: 2})
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func admitsAnotherPageOfHost(
	t *testing.T,
	limits *jetstream.PageVisitLimits,
	url canonicalurl.CanonicalURL,
	maxPages int,
) bool {
	t.Helper()
	admitted, err := limits.AdmitsAnotherPageOfHost(
		context.Background(), orderID, url, host, maxPages,
	)
	if err != nil {
		t.Fatalf("AdmitsAnotherPageOfHost: %v", err)
	}
	return admitted
}

func TestAHostTakesNoMorePagesThanItsProfileAllows(t *testing.T) {
	limits := pageVisitLimits(t)

	for page := range 2 {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/page%d", page))
		if !admitsAnotherPageOfHost(t, limits, url, 2) {
			t.Fatalf("page %d should be within the limit of the host", page+1)
		}
	}

	beyond := canonicalurltest.CanonicalURLOf(t, "http://host/beyond")
	if admitsAnotherPageOfHost(t, limits, beyond, 2) {
		t.Fatal("a page beyond the limit of the host should be refused")
	}
}

func TestAPageVisitTakesOnePageOfItsHostHoweverOftenItIsRedelivered(t *testing.T) {
	limits := pageVisitLimits(t)
	url := pageURL(t)

	for delivery := range 3 {
		if !admitsAnotherPageOfHost(t, limits, url, 1) {
			t.Fatalf("delivery %d should keep the page the page visit already holds", delivery)
		}
	}

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	if admitsAnotherPageOfHost(t, limits, other, 1) {
		t.Fatal("the redelivered page visit should hold the whole limit of the host")
	}
}

func TestAHostThatAllowsNoPagesAdmitsNone(t *testing.T) {
	if admitsAnotherPageOfHost(t, pageVisitLimits(t), pageURL(t), 0) {
		t.Fatal("a host that allows no pages should refuse the first page")
	}
}

func TestAnUnlimitedHostAdmitsEveryPage(t *testing.T) {
	limits := pageVisitLimits(t)

	for range 3 {
		url := pageURL(t)
		if !admitsAnotherPageOfHost(t, limits, url, yacycrawlcontract.UnlimitedPagesPerHost) {
			t.Fatal("an unlimited host should never refuse a page")
		}
	}
}

func TestTheSameHostCountsSeparatelyUnderAnotherOrder(t *testing.T) {
	limits := pageVisitLimits(t)
	url := pageURL(t)
	admitsAnotherPageOfHost(t, limits, url, 1)

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	admitted, err := limits.AdmitsAnotherPageOfHost(context.Background(), "o2", other, host, 1)
	if err != nil {
		t.Fatalf("AdmitsAnotherPageOfHost: %v", err)
	}
	if !admitted {
		t.Fatal("another order should hold its own limit of the host")
	}
}

func TestOnlyOneOfTwoConcurrentPageVisitsTakesTheLastPageOfAHost(t *testing.T) {
	limits := pageVisitLimits(t)
	ctx := context.Background()
	admissions := make(chan bool, 2)

	var taking sync.WaitGroup
	for page := range 2 {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/race%d", page))
		taking.Add(1)
		go func() {
			defer taking.Done()
			admitted, err := limits.AdmitsAnotherPageOfHost(ctx, orderID, url, host, 1)
			if err != nil {
				t.Errorf("AdmitsAnotherPageOfHost: %v", err)
				return
			}
			admissions <- admitted
		}()
	}
	taking.Wait()
	close(admissions)

	admitted := 0
	for granted := range admissions {
		if granted {
			admitted++
		}
	}
	if admitted != 1 {
		t.Fatalf("%d of two concurrent page visits took a host limit of one, want 1", admitted)
	}
}

func TestDeferralsRunOutAtTheirCeiling(t *testing.T) {
	limits := pageVisitLimits(t)
	ctx := context.Background()
	url := pageURL(t)

	for deferral := range 2 {
		admitted, err := limits.AdmitsAnotherDeferral(ctx, orderID, url)
		if err != nil {
			t.Fatalf("AdmitsAnotherDeferral: %v", err)
		}
		if !admitted {
			t.Fatalf("deferral %d should be within the ceiling", deferral+1)
		}
	}
	admitted, err := limits.AdmitsAnotherDeferral(ctx, orderID, url)
	if err != nil {
		t.Fatalf("AdmitsAnotherDeferral: %v", err)
	}
	if admitted {
		t.Fatal("a deferral beyond the ceiling should be refused")
	}
}

func TestEachAttemptReportsItsNumberUntilTheCeiling(t *testing.T) {
	limits := pageVisitLimits(t)
	ctx := context.Background()
	url := pageURL(t)

	for want := 1; want <= 2; want++ {
		attempt, admitted, err := limits.AdmitsAnotherAttempt(ctx, orderID, url)
		if err != nil {
			t.Fatalf("AdmitsAnotherAttempt: %v", err)
		}
		if !admitted || attempt != want {
			t.Fatalf("attempt %d admitted %v, want %d true", attempt, admitted, want)
		}
	}
	if _, admitted, err := limits.AdmitsAnotherAttempt(ctx, orderID, url); err != nil || admitted {
		t.Fatalf("an attempt beyond the ceiling should be refused, got %v %v", admitted, err)
	}
}

func TestDeferralsAndAttemptsKeepSeparateCeilings(t *testing.T) {
	limits := pageVisitLimits(t)
	ctx := context.Background()
	url := pageURL(t)

	for range 2 {
		if _, err := limits.AdmitsAnotherDeferral(ctx, orderID, url); err != nil {
			t.Fatalf("AdmitsAnotherDeferral: %v", err)
		}
	}
	_, admitted, err := limits.AdmitsAnotherAttempt(ctx, orderID, url)
	if err != nil {
		t.Fatalf("AdmitsAnotherAttempt: %v", err)
	}
	if !admitted {
		t.Fatal("spent deferrals should leave the attempts untouched")
	}
}

func TestTakingThePageOfAHostLeavesTheDeferralsUntouched(t *testing.T) {
	limits := pageVisitLimits(t)
	url := pageURL(t)
	admitsAnotherPageOfHost(t, limits, url, 1)

	admitted, err := limits.AdmitsAnotherDeferral(context.Background(), orderID, url)
	if err != nil {
		t.Fatalf("AdmitsAnotherDeferral: %v", err)
	}
	if !admitted {
		t.Fatal("a page visit holding the page of its host should still defer")
	}
}
