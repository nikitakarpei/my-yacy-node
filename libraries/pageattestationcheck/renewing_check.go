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

var ErrNoDeviceTrust = errors.New("the revocation list of attestation keys is not loaded")

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type TrustSource interface {
	Trust(ctx context.Context) (*android.Trust, error)
}

type TrustedReleases interface {
	Refresh(ctx context.Context, now time.Time)
	Current() ([]trustedappreleases.TrustedRelease, time.Time)
}

type RenewingCheck struct {
	current     atomic.Pointer[Check]
	trust       *android.Trust
	policy      Policy
	releases    TrustedReleases
	trustSource TrustSource
	clock       Clock
	observer    RenewalObserver
}

type RenewalSources struct {
	Releases TrustedReleases
	Trust    TrustSource
}

func NewRenewingCheck(
	policy Policy,
	sources RenewalSources,
	clock Clock,
	observer RenewalObserver,
) *RenewingCheck {
	check := &RenewingCheck{
		policy:      policy,
		releases:    sources.Releases,
		trustSource: sources.Trust,
		clock:       clock,
		observer:    observer,
	}
	check.current.Store(&Check{reason: ErrNoDeviceTrust})

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
	c.loadTrust(ctx)
	c.releases.Refresh(ctx, now)
	renewed := c.checkAt(now)
	c.current.Store(&renewed)
}

func (c *RenewingCheck) loadTrust(ctx context.Context) {
	trust, err := c.trustSource.Trust(ctx)
	if err != nil {
		c.observer.TrustUnloaded(ctx, err)

		return
	}
	c.trust = trust
}

func (c *RenewingCheck) checkAt(now time.Time) Check {
	if c.trust == nil {
		return Check{reason: ErrNoDeviceTrust}
	}
	releases, refreshedAt := c.releases.Current()

	return c.policy.CheckFor(releases, refreshedAt, c.trust, now)
}

func (c *RenewingCheck) VerdictOn(
	row yacymodel.URLMetadata,
	attestation attestedstatements.Bundle,
) Verdict {
	return c.current.Load().VerdictOn(row, attestation)
}
