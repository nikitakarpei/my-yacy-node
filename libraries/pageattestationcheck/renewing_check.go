package pageattestationcheck

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const renewalInterval = time.Hour

var ErrNoAndroidTrust = errors.New("the Android trust is not loaded")

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type AndroidTrustSource interface {
	Latest(ctx context.Context) (*android.Trust, error)
}

type TrustedReleases interface {
	Refresh(ctx context.Context, now time.Time)
	Current() ([]trustedappreleases.TrustedRelease, time.Time)
}

type RenewingCheck struct {
	current            atomic.Pointer[Check]
	androidTrust       *android.Trust
	policy             Policy
	releases           TrustedReleases
	androidTrustSource AndroidTrustSource
	clock              Clock
	observer           RenewalObserver
}

func NewRenewingCheck(
	policy Policy,
	releases TrustedReleases,
	androidTrustSource AndroidTrustSource,
	clock Clock,
	observer RenewalObserver,
) *RenewingCheck {
	check := &RenewingCheck{
		policy:             policy,
		releases:           releases,
		androidTrustSource: androidTrustSource,
		clock:              clock,
		observer:           observer,
	}
	check.current.Store(&Check{reason: ErrNoAndroidTrust})

	return check
}

func (c *RenewingCheck) Run(ctx context.Context) {
	for {
		c.renew(ctx)
		if err := c.clock.Sleep(ctx, renewalInterval); err != nil {
			return
		}
	}
}

func (c *RenewingCheck) renew(ctx context.Context) {
	now := c.clock.Now()
	c.loadAndroidTrust(ctx)
	c.releases.Refresh(ctx, now)
	renewed := c.checkAt(now)
	c.current.Store(&renewed)
}

func (c *RenewingCheck) loadAndroidTrust(ctx context.Context) {
	androidTrust, err := c.androidTrustSource.Latest(ctx)
	if err != nil {
		c.observer.AndroidTrustUnloaded(ctx, err)

		return
	}
	c.androidTrust = androidTrust
}

func (c *RenewingCheck) checkAt(now time.Time) Check {
	if c.androidTrust == nil {
		return Check{reason: ErrNoAndroidTrust}
	}
	releases, refreshedAt := c.releases.Current()

	return c.policy.CheckFor(releases, refreshedAt, c.androidTrust, now)
}

func (c *RenewingCheck) VerdictOn(
	row yacymodel.URLMetadata,
	attestation attestedstatements.Bundle,
) Verdict {
	return c.current.Load().VerdictOn(row, attestation)
}
