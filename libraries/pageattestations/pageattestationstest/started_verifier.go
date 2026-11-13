package pageattestationstest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/pageattestations"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

func StartedVerifier(
	t testing.TB,
	device *AttestingDevice,
	authority *TimeStampAuthority,
) *pageattestations.Verifier {
	t.Helper()
	clock := &firstRenewalClock{renewed: make(chan struct{})}
	pageVerifier := pageattestations.New(
		authority.Entry(),
		oneTrustedRelease{},
		deviceTrustSource{device: device},
		clock,
		unexpectedRenewalEvents{t: t},
	)
	go pageVerifier.Run(t.Context())
	<-clock.renewed

	return pageVerifier
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

type oneTrustedRelease struct{}

func (oneTrustedRelease) Refresh(context.Context, time.Time) {}

func (oneTrustedRelease) Releases() []trustedappreleases.TrustedRelease {
	return []trustedappreleases.TrustedRelease{{
		Tag:                "yacypeercompanion-v1.0.0",
		SigningCertificate: AppSigningCertificate,
		VersionCode:        AppVersionCode,
	}}
}

func (oneTrustedRelease) RefreshedAt() time.Time {
	return time.Now()
}

type deviceTrustSource struct {
	device *AttestingDevice
}

func (d deviceTrustSource) Load(context.Context) (*android.Trust, error) {
	return d.device.Trust(), nil
}

type unexpectedRenewalEvents struct {
	t testing.TB
}

func (u unexpectedRenewalEvents) AndroidTrustUnloaded(_ context.Context, cause error) {
	u.t.Errorf("the Android trust of the device is not loaded: %v", cause)
}

func (u unexpectedRenewalEvents) RenewalRefused(_ context.Context, reason error) {
	u.t.Errorf("the page attestation verifier is not renewed: %v", reason)
}
