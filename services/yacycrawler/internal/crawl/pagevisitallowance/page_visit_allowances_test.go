package pagevisitallowance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type fakeLimits struct {
	admitsPageOfHost bool
	admitsDeferral   bool
	admitsAttempt    bool
	attempt          int
	host             string
	err              error
}

func (limits *fakeLimits) AdmitsAnotherPageOfHost(
	_ context.Context, _ string, _ canonicalurl.CanonicalURL, host string, _ int,
) (bool, error) {
	limits.host = host
	return limits.admitsPageOfHost, limits.err
}

func (limits *fakeLimits) AdmitsAnotherDeferral(
	_ context.Context, _ string, _ canonicalurl.CanonicalURL,
) (bool, error) {
	return limits.admitsDeferral, limits.err
}

func (limits *fakeLimits) AdmitsAnotherAttempt(
	_ context.Context, _ string, _ canonicalurl.CanonicalURL,
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
		OrderID: "o1",
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
	}
}

func TestAHostPageIsGrantedWhenTheHostStillHasPagesLeft(t *testing.T) {
	limits := &fakeLimits{admitsPageOfHost: true}

	allowance, err := allowancesOf(limits).HostPageFor(t.Context(), pageVisitOf(t), 5)
	if err != nil {
		t.Fatalf("HostPageFor: %v", err)
	}

	if allowance.Disposal.DisposedThePage() {
		t.Fatalf("allowance = %+v, want a host with pages left to grant one", allowance)
	}
	if limits.host != "host" {
		t.Fatalf("took a page of %q, want the host of the visited url", limits.host)
	}
}

func TestAHostPageNamesTheLimitThatStoppedIt(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{}).HostPageFor(t.Context(), pageVisitOf(t), 5)
	if err != nil {
		t.Fatalf("HostPageFor: %v", err)
	}

	if allowance.Disposal != disposal.HostPagesExhausted {
		t.Fatalf("allowance = %+v, want host pages exhausted", allowance)
	}
}

func TestADeferralPausesForAsLongAsThePageVisitAsked(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{admitsDeferral: true}).
		DeferralFor(t.Context(), pageVisitOf(t), 90*time.Second)
	if err != nil {
		t.Fatalf("DeferralFor: %v", err)
	}

	if allowance.Disposal.DisposedThePage() || allowance.PauseFor != 90*time.Second {
		t.Fatalf("allowance = %+v, want a granted 90s pause", allowance)
	}
}

func TestADeferralNamesTheLimitThatStoppedIt(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{}).
		DeferralFor(t.Context(), pageVisitOf(t), time.Second)
	if err != nil {
		t.Fatalf("DeferralFor: %v", err)
	}

	if allowance.Disposal != disposal.DeferralsExhausted {
		t.Fatalf("allowance = %+v, want deferrals exhausted", allowance)
	}
}

func TestAnAttemptPausesForTheDelayOfItsAttemptNumber(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{attempt: 2, admitsAttempt: true}).
		AnotherAttemptFor(t.Context(), pageVisitOf(t))
	if err != nil {
		t.Fatalf("AnotherAttemptFor: %v", err)
	}

	want := retrydelay.Bounds{Floor: time.Second, Ceiling: time.Minute}.Delay(2)
	if allowance.Disposal.DisposedThePage() || allowance.PauseFor != want {
		t.Fatalf("allowance = %+v, want a granted pause of %s", allowance, want)
	}
}

func TestAnAttemptNamesTheLimitThatStoppedIt(t *testing.T) {
	allowance, err := allowancesOf(&fakeLimits{}).AnotherAttemptFor(t.Context(), pageVisitOf(t))
	if err != nil {
		t.Fatalf("AnotherAttemptFor: %v", err)
	}

	if allowance.Disposal != disposal.RetriesExhausted {
		t.Fatalf("allowance = %+v, want retries exhausted", allowance)
	}
}

func TestALimitThatCannotAnswerCarriesItsCauseBack(t *testing.T) {
	down := errors.New("bucket down")
	limits := &fakeLimits{err: down}

	if _, err := allowancesOf(limits).
		HostPageFor(t.Context(), pageVisitOf(t), 5); !errors.Is(err, down) {
		t.Fatalf("HostPageFor swallowed %v, got %v", down, err)
	}
	if _, err := allowancesOf(limits).
		DeferralFor(t.Context(), pageVisitOf(t), time.Second); !errors.Is(err, down) {
		t.Fatalf("DeferralFor swallowed %v, got %v", down, err)
	}
	if _, err := allowancesOf(limits).
		AnotherAttemptFor(t.Context(), pageVisitOf(t)); !errors.Is(err, down) {
		t.Fatalf("AnotherAttemptFor swallowed %v, got %v", down, err)
	}
}
