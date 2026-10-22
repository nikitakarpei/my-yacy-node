package pagevisitallowance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type fakeLimits struct {
	admitsPageOfHost bool
	admitsDeferral   bool
	admitsAttempt    bool
	attempt          int
	err              error
}

func (limits *fakeLimits) TakePageOfHost(
	_ context.Context, _ crawlid.CrawlID, _ canonicalurl.CanonicalURL, _ int,
) (bool, error) {
	return limits.admitsPageOfHost, limits.err
}

func (limits *fakeLimits) TakeDeferral(
	_ context.Context, _ crawlid.CrawlID, _ canonicalurl.CanonicalURL,
) (bool, error) {
	return limits.admitsDeferral, limits.err
}

func (limits *fakeLimits) TakeAttempt(
	_ context.Context, _ crawlid.CrawlID, _ canonicalurl.CanonicalURL,
) (int, bool, error) {
	return limits.attempt, limits.admitsAttempt, limits.err
}

func allowancesOf(limits *fakeLimits) *pagevisitallowance.Allowances {
	return pagevisitallowance.New(limits, retrydelay.Bounds{
		Floor:   time.Second,
		Ceiling: time.Minute,
	})
}

func pageVisitOf(t *testing.T) pendingpagevisit.PendingPageVisit {
	t.Helper()
	return pendingpagevisit.PendingPageVisit{
		CrawlID: crawlid.CrawlID{1},
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
	}
}

func TestAHostPageIsGrantedWhenTheHostStillHasPagesLeft(t *testing.T) {
	limits := &fakeLimits{admitsPageOfHost: true}

	allowance, err := allowancesOf(limits).TakePageOfHost(t.Context(), pageVisitOf(t), 5)
	if err != nil {
		t.Fatalf("TakePageOfHost: %v", err)
	}

	if !allowance.Granted {
		t.Fatalf("allowance = %+v, want a host with pages left to grant one", allowance)
	}
}

func TestAHostPageNamesTheLimitThatStoppedIt(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{}).TakePageOfHost(t.Context(), pageVisitOf(t), 5)
	if err != nil {
		t.Fatalf("TakePageOfHost: %v", err)
	}

	if allowance.Granted || allowance.DropReason != pagevisitdrop.HostPagesExhausted {
		t.Fatalf("allowance = %+v, want host pages exhausted", allowance)
	}
}

func TestADeferralPausesForAsLongAsThePageVisitAsked(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{admitsDeferral: true}).
		TakeDeferral(t.Context(), pageVisitOf(t), 90*time.Second)
	if err != nil {
		t.Fatalf("TakeDeferral: %v", err)
	}

	if !allowance.Granted || allowance.PauseFor != 90*time.Second {
		t.Fatalf("allowance = %+v, want a granted 90s pause", allowance)
	}
}

func TestADeferralNamesTheLimitThatStoppedIt(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{}).
		TakeDeferral(t.Context(), pageVisitOf(t), time.Second)
	if err != nil {
		t.Fatalf("TakeDeferral: %v", err)
	}

	if allowance.Granted || allowance.DropReason != pagevisitdrop.DeferralsExhausted {
		t.Fatalf("allowance = %+v, want deferrals exhausted", allowance)
	}
}

func TestAnAttemptPausesForTheDelayOfItsAttemptNumber(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{attempt: 2, admitsAttempt: true}).
		TakeAttempt(t.Context(), pageVisitOf(t))
	if err != nil {
		t.Fatalf("TakeAttempt: %v", err)
	}

	want := retrydelay.Bounds{Floor: time.Second, Ceiling: time.Minute}.Delay(2)
	if !allowance.Granted || allowance.PauseFor != want {
		t.Fatalf("allowance = %+v, want a granted pause of %s", allowance, want)
	}
}

func TestAnAttemptNamesTheLimitThatStoppedIt(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{}).TakeAttempt(t.Context(), pageVisitOf(t))
	if err != nil {
		t.Fatalf("TakeAttempt: %v", err)
	}

	if allowance.Granted || allowance.DropReason != pagevisitdrop.RetriesExhausted {
		t.Fatalf("allowance = %+v, want retries exhausted", allowance)
	}
}

func TestALimitThatCannotAnswerCarriesItsCauseBack(t *testing.T) {
	down := errors.New("bucket down")
	limits := &fakeLimits{err: down}

	if _, err := allowancesOf(limits).
		TakePageOfHost(t.Context(), pageVisitOf(t), 5); !errors.Is(err, down) {
		t.Fatalf("TakePageOfHost swallowed %v, got %v", down, err)
	}
	if _, err := allowancesOf(limits).
		TakeDeferral(t.Context(), pageVisitOf(t), time.Second); !errors.Is(err, down) {
		t.Fatalf("TakeDeferral swallowed %v, got %v", down, err)
	}
	if _, err := allowancesOf(limits).
		TakeAttempt(t.Context(), pageVisitOf(t)); !errors.Is(err, down) {
		t.Fatalf("TakeAttempt swallowed %v, got %v", down, err)
	}
}
