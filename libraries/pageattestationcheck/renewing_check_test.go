package pageattestationcheck_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

var (
	renewalEpoch       = time.Date(2026, time.November, 13, 12, 0, 0, 0, time.UTC)
	errListUnreachable = errors.New("the revocation list is unreachable")
)

func TestBeforeItsFirstRenewalTheCheckAttestsNoRow(t *testing.T) {
	renewal := newRenewal(t)
	row := rootedRow(t)

	verdict := renewal.check.VerdictOn(row, renewal.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestationcheck.ErrNoDeviceTrust) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestationcheck.ErrNoDeviceTrust)
	}
}

func TestARenewalTrustsTheRefreshedReleases(t *testing.T) {
	renewal := newRenewal(t)
	row := rootedRow(t)
	renewal.start(t)

	verdict := renewal.check.VerdictOn(row, renewal.attestationOf(t, row))

	if !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
	if refreshes := renewal.releases.refreshTimes(); len(refreshes) != 1 ||
		!refreshes[0].Equal(renewalEpoch) {
		t.Errorf("refreshes = %v, want one at %v", refreshes, renewalEpoch)
	}
}

func TestTheCheckRenewsAfterAnHour(t *testing.T) {
	renewal := newRenewal(t)
	row := rootedRow(t)
	sleep := renewal.start(t)
	renewal.releases.set(nil)

	renewal.clock.advanceBy(sleep)
	renewal.clock.wake()
	renewal.clock.sleepAfterRenewal()

	if sleep != time.Hour {
		t.Errorf("sleep = %v, want an hour", sleep)
	}
	verdict := renewal.check.VerdictOn(row, renewal.attestationOf(t, row))
	if !errors.Is(verdict.Reason, pageattestationcheck.ErrNoTrustedReleases) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestationcheck.ErrNoTrustedReleases)
	}
	if refreshes := renewal.releases.refreshTimes(); len(refreshes) != 2 ||
		!refreshes[1].Equal(renewalEpoch.Add(time.Hour)) {
		t.Errorf("refreshes = %v, want the second an hour after the first", refreshes)
	}
}

func TestARenewalThatCannotLoadTheListKeepsThePreviousList(t *testing.T) {
	renewal := newRenewal(t)
	row := rootedRow(t)
	renewal.start(t)
	renewal.trust.fail()

	renewal.clock.wake()
	renewal.clock.sleepAfterRenewal()

	if verdict := renewal.check.VerdictOn(row, renewal.attestationOf(t, row)); !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
	if !errors.Is(renewal.observer.cause(), errListUnreachable) {
		t.Errorf("observed cause = %v, want %v", renewal.observer.cause(), errListUnreachable)
	}
}

func TestAFirstRenewalThatCannotLoadTheListAttestsNoRow(t *testing.T) {
	renewal := newRenewal(t)
	row := rootedRow(t)
	renewal.trust.fail()
	renewal.start(t)

	verdict := renewal.check.VerdictOn(row, renewal.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestationcheck.ErrNoDeviceTrust) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestationcheck.ErrNoDeviceTrust)
	}
}

func TestTheCheckStopsRenewingWhenItsContextEnds(t *testing.T) {
	renewal := newRenewal(t)
	renewal.start(t)

	renewal.stop()

	<-renewal.stopped
}

type renewal struct {
	setup
	check    *pageattestationcheck.RenewingCheck
	clock    *fakeClock
	releases *fakeTrustedReleases
	trust    *fakeTrustSource
	observer *renewalEvents
	stop     context.CancelFunc
	stopped  chan struct{}
}

func newRenewal(t *testing.T) *renewal {
	t.Helper()
	setup := newSetup(t)
	renewal := &renewal{
		setup: setup,
		clock: &fakeClock{
			now:    renewalEpoch,
			sleeps: make(chan time.Duration),
			wakes:  make(chan struct{}),
		},
		releases: &fakeTrustedReleases{
			releases: []trustedappreleases.TrustedRelease{setup.trustedRelease()},
		},
		trust:    &fakeTrustSource{trust: setup.device.Trust()},
		observer: &renewalEvents{},
		stopped:  make(chan struct{}),
	}
	renewal.check = pageattestationcheck.NewRenewingCheck(
		setup.policy(t),
		pageattestationcheck.RenewalSources{Releases: renewal.releases, Trust: renewal.trust},
		renewal.clock,
		renewal.observer,
	)

	return renewal
}

func (r *renewal) start(t *testing.T) time.Duration {
	t.Helper()
	ctx, stop := context.WithCancel(t.Context())
	r.stop = stop
	t.Cleanup(stop)
	go func() {
		r.check.Run(ctx)
		close(r.stopped)
	}()

	return r.clock.sleepAfterRenewal()
}

type fakeClock struct {
	guard  sync.Mutex
	now    time.Time
	sleeps chan time.Duration
	wakes  chan struct{}
}

func (c *fakeClock) Now() time.Time {
	c.guard.Lock()
	defer c.guard.Unlock()

	return c.now
}

func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case c.sleeps <- d:
	case <-ctx.Done():
		return fmt.Errorf("context: %w", ctx.Err())
	}
	select {
	case <-c.wakes:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context: %w", ctx.Err())
	}
}

func (c *fakeClock) advanceBy(d time.Duration) {
	c.guard.Lock()
	defer c.guard.Unlock()

	c.now = c.now.Add(d)
}

func (c *fakeClock) sleepAfterRenewal() time.Duration {
	return <-c.sleeps
}

func (c *fakeClock) wake() {
	c.wakes <- struct{}{}
}

type fakeTrustedReleases struct {
	guard     sync.Mutex
	releases  []trustedappreleases.TrustedRelease
	refreshes []time.Time
}

func (r *fakeTrustedReleases) Refresh(_ context.Context, now time.Time) {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.refreshes = append(r.refreshes, now)
}

func (r *fakeTrustedReleases) Current() ([]trustedappreleases.TrustedRelease, time.Time) {
	r.guard.Lock()
	defer r.guard.Unlock()

	return r.releases, r.refreshes[len(r.refreshes)-1]
}

func (r *fakeTrustedReleases) set(releases []trustedappreleases.TrustedRelease) {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.releases = releases
}

func (r *fakeTrustedReleases) refreshTimes() []time.Time {
	r.guard.Lock()
	defer r.guard.Unlock()

	return append([]time.Time(nil), r.refreshes...)
}

type fakeTrustSource struct {
	guard sync.Mutex
	trust *android.Trust
}

func (s *fakeTrustSource) Trust(context.Context) (*android.Trust, error) {
	s.guard.Lock()
	defer s.guard.Unlock()
	if s.trust == nil {
		return nil, errListUnreachable
	}

	return s.trust, nil
}

func (s *fakeTrustSource) fail() {
	s.guard.Lock()
	defer s.guard.Unlock()

	s.trust = nil
}

type renewalEvents struct {
	guard         sync.Mutex
	observedCause error
}

func (e *renewalEvents) TrustUnloaded(_ context.Context, cause error) {
	e.guard.Lock()
	defer e.guard.Unlock()

	e.observedCause = cause
}

func (e *renewalEvents) cause() error {
	e.guard.Lock()
	defer e.guard.Unlock()

	return e.observedCause
}
