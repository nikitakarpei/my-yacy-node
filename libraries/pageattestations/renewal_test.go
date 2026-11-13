package pageattestations_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/pageattestations"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

var (
	renewalEpoch               = time.Date(2026, time.November, 13, 12, 0, 0, 0, time.UTC)
	errAndroidTrustUnreachable = errors.New("the Android trust is unreachable")
)

func TestBeforeItsFirstRenewalTheCheckAttestsNoRow(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)

	verdict := renewal.pageVerifier.VerdictOn(row, renewal.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestations.ErrNoAndroidTrust) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestations.ErrNoAndroidTrust)
	}
}

func TestARenewalTrustsTheRefreshedReleases(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.start(t)

	verdict := renewal.pageVerifier.VerdictOn(row, renewal.attestationOf(t, row))

	if !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
	if refreshes := renewal.trustedReleases.refreshTimes(); len(refreshes) != 1 ||
		!refreshes[0].Equal(renewalEpoch) {
		t.Errorf("refreshes = %v, want one at %v", refreshes, renewalEpoch)
	}
}

func TestTheCheckRenewsAfterAnHour(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	sleep := renewal.start(t)
	renewal.trustedReleases.set(nil)

	renewal.clock.advanceBy(sleep)
	renewal.clock.wake()
	renewal.clock.sleepAfterRenewal()

	if sleep != time.Hour {
		t.Errorf("sleep = %v, want an hour", sleep)
	}
	verdict := renewal.pageVerifier.VerdictOn(row, renewal.attestationOf(t, row))
	if !errors.Is(verdict.Reason, pageattestations.ErrNoTrustedReleases) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestations.ErrNoTrustedReleases)
	}
	if refreshes := renewal.trustedReleases.refreshTimes(); len(refreshes) != 2 ||
		!refreshes[1].Equal(renewalEpoch.Add(time.Hour)) {
		t.Errorf("refreshes = %v, want the second an hour after the first", refreshes)
	}
}

func TestARenewalThatCannotLoadTheAndroidTrustKeepsThePreviousOne(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.start(t)
	renewal.androidTrustSource.fail()

	renewal.clock.wake()
	renewal.clock.sleepAfterRenewal()

	if verdict := renewal.pageVerifier.VerdictOn(
		row,
		renewal.attestationOf(t, row),
	); !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
	if !errors.Is(renewal.observer.cause(), errAndroidTrustUnreachable) {
		t.Errorf(
			"observed cause = %v, want %v",
			renewal.observer.cause(),
			errAndroidTrustUnreachable,
		)
	}
}

func TestAFirstRenewalThatCannotLoadTheAndroidTrustAttestsNoRow(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.androidTrustSource.fail()
	renewal.start(t)

	verdict := renewal.pageVerifier.VerdictOn(row, renewal.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestations.ErrNoAndroidTrust) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestations.ErrNoAndroidTrust)
	}
}

func TestARenewalThatCannotLoadTheAndroidTrustTakesTheRefreshedReleases(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.start(t)
	renewal.androidTrustSource.fail()
	renewal.trustedReleases.set(nil)

	renewal.clock.wake()
	renewal.clock.sleepAfterRenewal()

	verdict := renewal.pageVerifier.VerdictOn(row, renewal.attestationOf(t, row))
	if !errors.Is(verdict.Reason, pageattestations.ErrNoTrustedReleases) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestations.ErrNoTrustedReleases)
	}
}

func TestReleasesThatFailToRefreshForTwoDaysAttestNoRow(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.start(t)
	renewal.trustedReleases.failRefreshes()

	renewal.clock.advanceBy(49 * time.Hour)
	renewal.clock.wake()
	renewal.clock.sleepAfterRenewal()

	verdict := renewal.pageVerifier.VerdictOn(row, renewal.attestationOf(t, row))
	if !errors.Is(verdict.Reason, pageattestations.ErrStaleTrustedReleases) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestations.ErrStaleTrustedReleases)
	}
	if !errors.Is(renewal.observer.refusal(), pageattestations.ErrStaleTrustedReleases) {
		t.Errorf("refusal = %v, want %v",
			renewal.observer.refusal(), pageattestations.ErrStaleTrustedReleases)
	}
}

func TestTheCheckStopsRenewingWhenItsContextEnds(t *testing.T) {
	renewal := newRenewingVerifier(t)
	renewal.start(t)

	renewal.stop()

	<-renewal.stopped
}

type renewingVerifier struct {
	setup
	pageVerifier       *pageattestations.Verifier
	clock              *fakeClock
	trustedReleases    *fakeTrustedReleases
	androidTrustSource *fakeAndroidTrustSource
	observer           *renewalEvents
	stop               context.CancelFunc
	stopped            chan struct{}
}

func newRenewingVerifier(t *testing.T) *renewingVerifier {
	t.Helper()
	setup := newSetup(t)

	return setup.renewingVerifierOf(t, setup.trustedRelease())
}

func (s setup) renewingVerifierOf(
	t *testing.T,
	releases ...trustedappreleases.TrustedRelease,
) *renewingVerifier {
	t.Helper()
	renewal := &renewingVerifier{
		setup: s,
		clock: &fakeClock{
			now:    renewalEpoch,
			sleeps: make(chan time.Duration),
			wakes:  make(chan struct{}),
		},
		trustedReleases:    &fakeTrustedReleases{releases: releases},
		androidTrustSource: &fakeAndroidTrustSource{androidTrust: s.device.Trust()},
		observer:           &renewalEvents{},
		stopped:            make(chan struct{}),
	}
	renewal.pageVerifier = pageattestations.New(
		s.authority.Entry(),
		renewal.trustedReleases,
		renewal.androidTrustSource,
		renewal.clock,
		renewal.observer,
	)

	return renewal
}

func (r *renewingVerifier) start(t *testing.T) time.Duration {
	t.Helper()
	ctx, stop := context.WithCancel(t.Context())
	r.stop = stop
	t.Cleanup(stop)
	go func() {
		r.pageVerifier.Run(ctx)
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
	guard       sync.Mutex
	releases    []trustedappreleases.TrustedRelease
	refreshes   []time.Time
	refreshedAt time.Time
	failing     bool
}

func (r *fakeTrustedReleases) Refresh(_ context.Context, now time.Time) {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.refreshes = append(r.refreshes, now)
	if !r.failing {
		r.refreshedAt = now
	}
}

func (r *fakeTrustedReleases) Releases() []trustedappreleases.TrustedRelease {
	r.guard.Lock()
	defer r.guard.Unlock()

	return r.releases
}

func (r *fakeTrustedReleases) RefreshedAt() time.Time {
	r.guard.Lock()
	defer r.guard.Unlock()

	return r.refreshedAt
}

func (r *fakeTrustedReleases) failRefreshes() {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.failing = true
}

func (r *fakeTrustedReleases) failRefreshesSince(age time.Duration) {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.failing = true
	r.refreshedAt = renewalEpoch.Add(-age)
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

type fakeAndroidTrustSource struct {
	guard        sync.Mutex
	androidTrust *android.Trust
}

func (s *fakeAndroidTrustSource) Load(context.Context) (*android.Trust, error) {
	s.guard.Lock()
	defer s.guard.Unlock()
	if s.androidTrust == nil {
		return nil, errAndroidTrustUnreachable
	}

	return s.androidTrust, nil
}

func (s *fakeAndroidTrustSource) fail() {
	s.guard.Lock()
	defer s.guard.Unlock()

	s.androidTrust = nil
}

type renewalEvents struct {
	guard         sync.Mutex
	observedCause error
	refusedReason error
}

func (e *renewalEvents) AndroidTrustUnloaded(_ context.Context, cause error) {
	e.guard.Lock()
	defer e.guard.Unlock()

	e.observedCause = cause
}

func (e *renewalEvents) cause() error {
	e.guard.Lock()
	defer e.guard.Unlock()

	return e.observedCause
}

func (e *renewalEvents) RenewalRefused(_ context.Context, reason error) {
	e.guard.Lock()
	defer e.guard.Unlock()

	e.refusedReason = reason
}

func (e *renewalEvents) refusal() error {
	e.guard.Lock()
	defer e.guard.Unlock()

	return e.refusedReason
}
