package pageattestationcheck

import (
	"context"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/companionapp"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	drandChain            = "quicknet"
	maxTrustedReleasesAge = 48 * time.Hour
	renewalInterval       = time.Hour
)

var (
	ErrNoPageRWIRoot      = errors.New("the row has no page RWI root")
	ErrOtherPageRWIRoot   = errors.New("the attestation is of another page RWI root")
	ErrOtherAddress       = errors.New("the attestation is of another address")
	ErrHashOfOtherAddress = errors.New(
		"the URL hash of the row is not the hash of its address",
	)
	ErrNoTrustedReleases    = errors.New("no app release is trusted")
	ErrStaleTrustedReleases = errors.New(
		"the trusted app releases are older than their maximum age",
	)
	ErrNoAndroidTrust = errors.New("the Android trust is not loaded")
)

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

type Check struct {
	lastRenewal        atomic.Pointer[renewal]
	androidTrust       *android.Trust
	timeStampAuthority verifier.TSAEntry
	releases           TrustedReleases
	androidTrustSource AndroidTrustSource
	clock              Clock
	observer           RenewalObserver
}

type renewal struct {
	statementVerifier *verifier.Verifier
	reason            error
}

type Verdict struct {
	Attested  bool
	FetchedAt time.Time
	Reason    error
}

func New(
	timeStampAuthority verifier.TSAEntry,
	releases TrustedReleases,
	androidTrustSource AndroidTrustSource,
	clock Clock,
	observer RenewalObserver,
) *Check {
	check := &Check{
		timeStampAuthority: timeStampAuthority,
		releases:           releases,
		androidTrustSource: androidTrustSource,
		clock:              clock,
		observer:           observer,
	}
	check.lastRenewal.Store(&renewal{reason: ErrNoAndroidTrust})

	return check
}

func (c *Check) Run(ctx context.Context) {
	for {
		c.renew(ctx)
		if err := c.clock.Sleep(ctx, renewalInterval); err != nil {
			return
		}
	}
}

func (c *Check) renew(ctx context.Context) {
	now := c.clock.Now()
	c.loadAndroidTrust(ctx)
	c.releases.Refresh(ctx, now)
	renewed := c.renewalAt(now)
	c.lastRenewal.Store(&renewed)
}

func (c *Check) loadAndroidTrust(ctx context.Context) {
	androidTrust, err := c.androidTrustSource.Latest(ctx)
	if err != nil {
		c.observer.AndroidTrustUnloaded(ctx, err)

		return
	}
	c.androidTrust = androidTrust
}

func (c *Check) renewalAt(now time.Time) renewal {
	releases, refreshedAt := c.releases.Current()
	switch {
	case c.androidTrust == nil:
		return renewal{reason: ErrNoAndroidTrust}
	case len(releases) == 0:
		return renewal{reason: ErrNoTrustedReleases}
	case now.Sub(refreshedAt) > maxTrustedReleasesAge:
		return renewal{reason: fmt.Errorf(
			"%w: refreshed at %s", ErrStaleTrustedReleases, refreshedAt.Format(time.RFC3339),
		)}
	}
	statementVerifier, err := verifier.New(
		policyFor(c.timeStampAuthority, releases), android.New(c.androidTrust),
	)
	if err != nil {
		return renewal{reason: fmt.Errorf("page attestation verifier: %w", err)}
	}

	return renewal{statementVerifier: statementVerifier}
}

func policyFor(
	timeStampAuthority verifier.TSAEntry,
	releases []trustedappreleases.TrustedRelease,
) *verifier.Policy {
	return &verifier.Policy{
		Policy:               verifier.PolicyVersion,
		TimestampAuthorities: []verifier.TSAEntry{timeStampAuthority},
		DrandChains:          []string{drandChain},
		PredicateTypes: map[string]verifier.PredicatePolicy{
			pagerwicontract.PageRWIPredicateType: {
				TrustedSigners:      signersFor(releases),
				RequireTimestamp:    true,
				RequireStartedAfter: true,
			},
		},
	}
}

func signersFor(releases []trustedappreleases.TrustedRelease) []jsontext.Value {
	signers := make([]jsontext.Value, 0, len(releases))
	for _, release := range releases {
		signer := android.Signer{
			Profile:             android.Name,
			Package:             companionapp.Package,
			SigningCertificates: []string{release.SigningCertificate},
			VersionCodes:        []int64{release.VersionCode},
		}
		if !release.EndTime.IsZero() {
			endTime := release.EndTime.Unix()
			signer.VersionEndTimes = []android.VersionEndTime{
				{VersionCode: &release.VersionCode, EndTime: &endTime},
			}
		}
		encoding, _ := json.Marshal(signer) //nolint:errchkjson // strings and numbers.
		signers = append(signers, encoding)
	}

	return signers
}

func (c *Check) VerdictOn(
	row yacymodel.URLMetadata,
	attestation attestedstatements.Bundle,
) Verdict {
	lastRenewal := c.lastRenewal.Load()
	if lastRenewal.reason != nil {
		return notAttested(lastRenewal.reason)
	}
	bundle, err := verifier.ParseBundle(attestation)
	if err != nil {
		return notAttested(fmt.Errorf("read the attestation: %w", err))
	}
	statement, err := pagerwicontract.UnmarshalPageRWIStatement(bundle.DSSEEnvelope.Payload)
	if err != nil {
		return notAttested(err)
	}
	if err := reasonAgainstRow(row, statement); err != nil {
		return notAttested(err)
	}
	result, err := lastRenewal.statementVerifier.Verify(bundle)
	if err != nil {
		return notAttested(fmt.Errorf("verify the attestation: %w", err))
	}

	return Verdict{Attested: true, FetchedAt: *result.StartedAfter}
}

func notAttested(reason error) Verdict {
	return Verdict{Reason: reason}
}

func reasonAgainstRow(
	row yacymodel.URLMetadata,
	statement pagerwicontract.PageRWIStatement,
) error {
	root, rooted := yacymodel.PageRWIRootFrom(row).Get()
	address, err := url.Parse(row.Address)
	switch {
	case !rooted:
		return ErrNoPageRWIRoot
	case root != statement.PageRWIRoot:
		return ErrOtherPageRWIRoot
	case statement.PageURL.String() != row.Address:
		return ErrOtherAddress
	case err != nil || yacymodel.URLHashOf(address) != row.Hash:
		return ErrHashOfOtherAddress
	default:
		return nil
	}
}
