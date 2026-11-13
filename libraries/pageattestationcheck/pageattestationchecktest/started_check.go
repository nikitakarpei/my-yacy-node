package pageattestationchecktest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

func StartedCheck(
	t testing.TB,
	device *AttestingDevice,
	authority *TimeStampAuthority,
) *pageattestationcheck.Check {
	t.Helper()
	clock := &firstRenewalClock{renewed: make(chan struct{})}
	check := pageattestationcheck.New(
		authority.Entry(), appRelease{}, deviceTrust{device: device}, clock, unloadedTrust{t: t},
	)
	go check.Run(t.Context())
	<-clock.renewed

	return check
}

type firstRenewalClock struct {
	once    sync.Once
	renewed chan struct{}
}

func (c *firstRenewalClock) Now() time.Time {
	return time.Now()
}

func (c *firstRenewalClock) Sleep(ctx context.Context, _ time.Duration) error {
	c.once.Do(func() { close(c.renewed) })
	<-ctx.Done()

	return fmt.Errorf("context: %w", ctx.Err())
}

type appRelease struct{}

func (appRelease) Refresh(context.Context, time.Time) {}

func (appRelease) Current() ([]trustedappreleases.TrustedRelease, time.Time) {
	return []trustedappreleases.TrustedRelease{{
		Tag:                "yacypeercompanion-v1.0.0",
		SigningCertificate: AppSigningCertificate,
		VersionCode:        AppVersionCode,
	}}, time.Now()
}

type deviceTrust struct {
	device *AttestingDevice
}

func (d deviceTrust) Latest(context.Context) (*android.Trust, error) {
	return d.device.Trust(), nil
}

type unloadedTrust struct {
	t testing.TB
}

func (u unloadedTrust) AndroidTrustUnloaded(_ context.Context, cause error) {
	u.t.Errorf("the Android trust of the device is not loaded: %v", cause)
}
