package pageattestations_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestations"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var (
	renewalEpoch               = time.Date(2026, time.November, 13, 12, 0, 0, 0, time.UTC)
	errAndroidTrustUnreachable = errors.New("the Android trust is unreachable")
)

func TestBeforeItsFirstRenewalTheVerifierAttestsNoRow(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)

	verdict := renewal.verdictOn(t, row, renewal.attestationOf(t, row))

	if verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
}

func TestARenewalTrustsTheRefreshedReleases(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.start(t)

	verdict := renewal.verdictOn(t, row, renewal.attestationOf(t, row))

	if !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
	if refreshes := renewal.trustedReleases.refreshTimes(); len(refreshes) != 1 ||
		!refreshes[0].Equal(renewalEpoch) {
		t.Errorf("refreshes = %v, want one at %v", refreshes, renewalEpoch)
	}
}

func TestTheVerifierRenewsAfterAnHour(t *testing.T) {
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
	if verdict := renewal.verdictOn(t, row, renewal.attestationOf(t, row)); verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
	if !errors.Is(renewal.observer.noSignerReason(), pageattestations.ErrNoTrustedReleases) {
		t.Errorf("no signer reason = %v, want %v",
			renewal.observer.noSignerReason(), pageattestations.ErrNoTrustedReleases)
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

	if verdict := renewal.verdictOn(t, row, renewal.attestationOf(t, row)); !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
	if !errors.Is(renewal.observer.androidTrustCause(), errAndroidTrustUnreachable) {
		t.Errorf(
			"observed cause = %v, want %v",
			renewal.observer.androidTrustCause(),
			errAndroidTrustUnreachable,
		)
	}
}

func TestAFirstRenewalThatCannotLoadTheAndroidTrustAttestsNoRow(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.androidTrustSource.fail()
	renewal.start(t)

	verdict := renewal.verdictOn(t, row, renewal.attestationOf(t, row))

	if verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
	if !errors.Is(renewal.observer.noSignerReason(), pageattestations.ErrNoAndroidTrust) {
		t.Errorf("no signer reason = %v, want %v",
			renewal.observer.noSignerReason(), pageattestations.ErrNoAndroidTrust)
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

	if verdict := renewal.verdictOn(t, row, renewal.attestationOf(t, row)); verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
	if !errors.Is(renewal.observer.noSignerReason(), pageattestations.ErrNoTrustedReleases) {
		t.Errorf("no signer reason = %v, want %v",
			renewal.observer.noSignerReason(), pageattestations.ErrNoTrustedReleases)
	}
}

func TestDenialsUnreadForTwoDaysAttestNoRow(t *testing.T) {
	renewal := newRenewingVerifier(t)
	row := rootedRow(t)
	renewal.start(t)
	renewal.trustedReleases.failDenialReads()

	renewal.clock.advanceBy(49 * time.Hour)
	renewal.clock.wake()
	renewal.clock.sleepAfterRenewal()

	if verdict := renewal.verdictOn(t, row, renewal.attestationOf(t, row)); verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
	if !errors.Is(renewal.observer.noSignerReason(), pageattestations.ErrStaleReleaseDenials) {
		t.Errorf("no signer reason = %v, want %v",
			renewal.observer.noSignerReason(), pageattestations.ErrStaleReleaseDenials)
	}
}

func TestTheVerifierStopsRenewingWhenItsContextEnds(t *testing.T) {
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
	observer           *verificationEvents
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
		observer:           &verificationEvents{},
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

func (r *renewingVerifier) verdictOn(
	t *testing.T,
	row yacymodel.URLMetadata,
	attestation attestedstatements.Bundle,
) pageattestations.Verdict {
	t.Helper()

	return r.pageVerifier.VerdictOn(t.Context(), row, attestation)
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
	guard         sync.Mutex
	releases      []trustedappreleases.TrustedRelease
	refreshes     []time.Time
	denialsReadAt time.Time
	denialsUnread bool
}

func (r *fakeTrustedReleases) Refresh(_ context.Context, now time.Time) {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.refreshes = append(r.refreshes, now)
	if !r.denialsUnread {
		r.denialsReadAt = now
	}
}

func (r *fakeTrustedReleases) Releases() []trustedappreleases.TrustedRelease {
	r.guard.Lock()
	defer r.guard.Unlock()

	return r.releases
}

func (r *fakeTrustedReleases) DenialsReadAt() time.Time {
	r.guard.Lock()
	defer r.guard.Unlock()

	return r.denialsReadAt
}

func (r *fakeTrustedReleases) failDenialReads() {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.denialsUnread = true
}

func (r *fakeTrustedReleases) failDenialReadsSince(age time.Duration) {
	r.guard.Lock()
	defer r.guard.Unlock()

	r.denialsUnread = true
	r.denialsReadAt = renewalEpoch.Add(-age)
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

type verificationEvents struct {
	guard             sync.Mutex
	unloadedCause     error
	untrustedReason   error
	notAttestedReason error
}

func (e *verificationEvents) AndroidTrustUnloaded(_ context.Context, cause error) {
	e.guard.Lock()
	defer e.guard.Unlock()

	e.unloadedCause = cause
}

func (e *verificationEvents) androidTrustCause() error {
	e.guard.Lock()
	defer e.guard.Unlock()

	return e.unloadedCause
}

func (e *verificationEvents) NoSignerTrusted(_ context.Context, reason error) {
	e.guard.Lock()
	defer e.guard.Unlock()

	e.untrustedReason = reason
}

func (e *verificationEvents) noSignerReason() error {
	e.guard.Lock()
	defer e.guard.Unlock()

	return e.untrustedReason
}

func (e *verificationEvents) RowNotAttested(_ context.Context, _ string, reason error) {
	e.guard.Lock()
	defer e.guard.Unlock()

	e.notAttestedReason = reason
}

func (e *verificationEvents) rowReason() error {
	e.guard.Lock()
	defer e.guard.Unlock()

	return e.notAttestedReason
}
