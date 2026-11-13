package pageattestations

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
	Load(ctx context.Context) (*android.Trust, error)
}

type TrustedReleases interface {
	Refresh(ctx context.Context, now time.Time)
	Releases() []trustedappreleases.TrustedRelease
	RefreshedAt() time.Time
}

type Verifier struct {
	statementVerifier  atomic.Pointer[verifier.Verifier]
	androidTrust       *android.Trust
	timeStampAuthority verifier.TSAEntry
	trustedReleases    TrustedReleases
	androidTrustSource AndroidTrustSource
	clock              Clock
	observer           VerificationObserver
}

type Verdict struct {
	Attested  bool
	FetchedAt time.Time
}

func New(
	timeStampAuthority verifier.TSAEntry,
	trustedReleases TrustedReleases,
	androidTrustSource AndroidTrustSource,
	clock Clock,
	observer VerificationObserver,
) *Verifier {
	return &Verifier{
		timeStampAuthority: timeStampAuthority,
		trustedReleases:    trustedReleases,
		androidTrustSource: androidTrustSource,
		clock:              clock,
		observer:           observer,
	}
}

func (v *Verifier) Run(ctx context.Context) {
	for {
		v.renew(ctx)
		if err := v.clock.Sleep(ctx, renewalInterval); err != nil {
			return
		}
	}
}

func (v *Verifier) renew(ctx context.Context) {
	now := v.clock.Now()
	v.loadAndroidTrust(ctx)
	v.trustedReleases.Refresh(ctx, now)
	statementVerifier, err := v.statementVerifierAt(now)
	if err != nil {
		v.observer.NoSignerTrusted(ctx, err)
	}
	v.statementVerifier.Store(statementVerifier)
}

func (v *Verifier) loadAndroidTrust(ctx context.Context) {
	androidTrust, err := v.androidTrustSource.Load(ctx)
	if err != nil {
		v.observer.AndroidTrustUnloaded(ctx, err)

		return
	}
	v.androidTrust = androidTrust
}

func (v *Verifier) statementVerifierAt(now time.Time) (*verifier.Verifier, error) {
	releases, refreshedAt := v.trustedReleases.Releases(), v.trustedReleases.RefreshedAt()
	switch {
	case v.androidTrust == nil:
		return nil, ErrNoAndroidTrust
	case len(releases) == 0:
		return nil, ErrNoTrustedReleases
	case now.Sub(refreshedAt) > maxTrustedReleasesAge:
		return nil, fmt.Errorf(
			"%w: refreshed at %s", ErrStaleTrustedReleases, refreshedAt.Format(time.RFC3339),
		)
	}
	statementVerifier, err := verifier.New(
		policyFor(v.timeStampAuthority, releases), android.New(v.androidTrust),
	)
	if err != nil {
		return nil, fmt.Errorf("make the statement verifier: %w", err)
	}

	return statementVerifier, nil
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

func (v *Verifier) VerdictOn(
	ctx context.Context,
	row yacymodel.URLMetadata,
	attestation attestedstatements.Bundle,
) Verdict {
	statementVerifier := v.statementVerifier.Load()
	if statementVerifier == nil {
		return Verdict{}
	}
	fetchedAt, err := fetchTimeOf(attestation, row, statementVerifier)
	if err != nil {
		v.observer.RowNotAttested(ctx, row.Address, err)

		return Verdict{}
	}

	return Verdict{Attested: true, FetchedAt: fetchedAt}
}

func fetchTimeOf(
	attestation attestedstatements.Bundle,
	row yacymodel.URLMetadata,
	statementVerifier *verifier.Verifier,
) (time.Time, error) {
	bundle, err := verifier.ParseBundle(attestation)
	if err != nil {
		return time.Time{}, fmt.Errorf("read the attestation: %w", err)
	}
	statement, err := pagerwicontract.UnmarshalPageRWIStatement(bundle.DSSEEnvelope.Payload)
	if err != nil {
		return time.Time{}, fmt.Errorf("read the statement: %w", err)
	}
	if err := reasonAgainstRow(row, statement); err != nil {
		return time.Time{}, err
	}
	result, err := statementVerifier.Verify(bundle)
	if err != nil {
		return time.Time{}, fmt.Errorf("verify the attestation: %w", err)
	}

	return *result.StartedAfter, nil
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
