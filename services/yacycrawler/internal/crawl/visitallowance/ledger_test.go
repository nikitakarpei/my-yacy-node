package visitallowance_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitallowance"
)

type fakeClaims struct {
	deferred bool
	attempt  int
	retried  bool
	err      error
}

func (c *fakeClaims) Defer(_ context.Context, _ string, _ canonicalurl.CanonicalURL) (bool, error) {
	return c.deferred, c.err
}

func (c *fakeClaims) Retry(
	_ context.Context, _ string, _ canonicalurl.CanonicalURL,
) (int, bool, error) {
	return c.attempt, c.retried, c.err
}

type fakeHostPageAllowances struct {
	spent bool
	host  string
	err   error
}

func (a *fakeHostPageAllowances) HoldsHostPage(
	_ context.Context, _ string, _ canonicalurl.CanonicalURL, host string, _ int,
) (bool, error) {
	a.host = host
	return a.spent, a.err
}

func ledgerOf(claims *fakeClaims, hostPages *fakeHostPageAllowances) *visitallowance.Ledger {
	return visitallowance.New(claims, hostPages, retrydelay.Bounds{
		Floor:   time.Second,
		Ceiling: time.Minute,
	})
}

func visitOf(t *testing.T) pendingvisit.PendingVisit {
	t.Helper()
	return pendingvisit.PendingVisit{
		OrderID: "o1",
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
	}
}

func TestAHostPageIsGrantedWhenTheHostStillHasPagesToSpend(t *testing.T) {
	hostPages := &fakeHostPageAllowances{spent: true}

	allowance, err := ledgerOf(&fakeClaims{}, hostPages).
		HostPageFor(t.Context(), visitOf(t), 5)
	if err != nil {
		t.Fatalf("HostPageFor: %v", err)
	}

	if !allowance.Granted {
		t.Fatal("a host with pages left should grant one")
	}
	if hostPages.host != "host" {
		t.Fatalf("spent a page of %q, want the host of the visited url", hostPages.host)
	}
}

func TestAHostPageNamesTheAllowanceThatRanOut(t *testing.T) {
	allowance, err := ledgerOf(&fakeClaims{}, &fakeHostPageAllowances{}).
		HostPageFor(t.Context(), visitOf(t), 5)
	if err != nil {
		t.Fatalf("HostPageFor: %v", err)
	}

	if allowance.Granted || allowance.Exhausted != disposal.HostPagesExhausted {
		t.Fatalf("allowance = %+v, want host pages exhausted", allowance)
	}
}

func TestADeferralPausesForAsLongAsTheVisitAsked(t *testing.T) {
	allowance, err := ledgerOf(&fakeClaims{deferred: true}, &fakeHostPageAllowances{}).
		DeferralFor(t.Context(), visitOf(t), 90*time.Second)
	if err != nil {
		t.Fatalf("DeferralFor: %v", err)
	}

	if !allowance.Granted || allowance.PauseFor != 90*time.Second {
		t.Fatalf("allowance = %+v, want a granted 90s pause", allowance)
	}
}

func TestADeferralNamesTheAllowanceThatRanOut(t *testing.T) {
	allowance, err := ledgerOf(&fakeClaims{}, &fakeHostPageAllowances{}).
		DeferralFor(t.Context(), visitOf(t), time.Second)
	if err != nil {
		t.Fatalf("DeferralFor: %v", err)
	}

	if allowance.Granted || allowance.Exhausted != disposal.DeferralsExhausted {
		t.Fatalf("allowance = %+v, want deferrals exhausted", allowance)
	}
}

func TestAnAttemptPausesForTheDelayOfItsAttemptNumber(t *testing.T) {
	allowance, err := ledgerOf(&fakeClaims{attempt: 2, retried: true}, &fakeHostPageAllowances{}).
		AnotherAttemptFor(t.Context(), visitOf(t))
	if err != nil {
		t.Fatalf("AnotherAttemptFor: %v", err)
	}

	want := retrydelay.Bounds{Floor: time.Second, Ceiling: time.Minute}.Delay(2)
	if !allowance.Granted || allowance.PauseFor != want {
		t.Fatalf("allowance = %+v, want a granted pause of %s", allowance, want)
	}
}

func TestAnAttemptNamesTheAllowanceThatRanOut(t *testing.T) {
	allowance, err := ledgerOf(&fakeClaims{}, &fakeHostPageAllowances{}).
		AnotherAttemptFor(t.Context(), visitOf(t))
	if err != nil {
		t.Fatalf("AnotherAttemptFor: %v", err)
	}

	if allowance.Granted || allowance.Exhausted != disposal.RetriesExhausted {
		t.Fatalf("allowance = %+v, want retries exhausted", allowance)
	}
}

func TestAFailedSpendCarriesItsCauseBack(t *testing.T) {
	down := errors.New("bucket down")

	if _, err := ledgerOf(&fakeClaims{}, &fakeHostPageAllowances{err: down}).
		HostPageFor(t.Context(), visitOf(t), 5); !errors.Is(err, down) {
		t.Fatalf("HostPageFor swallowed %v, got %v", down, err)
	}
	if _, err := ledgerOf(&fakeClaims{err: down}, &fakeHostPageAllowances{}).
		DeferralFor(t.Context(), visitOf(t), time.Second); !errors.Is(err, down) {
		t.Fatalf("DeferralFor swallowed %v, got %v", down, err)
	}
	if _, err := ledgerOf(&fakeClaims{err: down}, &fakeHostPageAllowances{}).
		AnotherAttemptFor(t.Context(), visitOf(t)); !errors.Is(err, down) {
		t.Fatalf("AnotherAttemptFor swallowed %v, got %v", down, err)
	}
}
