package pageattestations

import (
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"net/url"
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
	drandChain           = "quicknet"
	maxReleaseDenialsAge = 48 * time.Hour
)

var (
	ErrNoPageRWIRoot      = errors.New("the row has no page RWI root")
	ErrOtherPageRWIRoot   = errors.New("the attestation is of another page RWI root")
	ErrOtherAddress       = errors.New("the attestation is of another address")
	ErrHashOfOtherAddress = errors.New(
		"the URL hash of the row is not the hash of its address",
	)
	ErrNoTrustedReleases   = errors.New("no app release is trusted")
	ErrStaleReleaseDenials = errors.New(
		"the release denials are older than their maximum age",
	)
)

type Verifier struct {
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
	releases []trustedappreleases.TrustedRelease,
	denialsReadAt time.Time,
	androidTrust *android.Trust,
	now time.Time,
) Verifier {
	switch {
	case len(releases) == 0:
		return Verifier{reason: ErrNoTrustedReleases}
	case now.Sub(denialsReadAt) > maxReleaseDenialsAge:
		return Verifier{reason: fmt.Errorf(
			"%w: read at %s", ErrStaleReleaseDenials, denialsReadAt.Format(time.RFC3339),
		)}
	}
	statementVerifier, err := verifier.New(
		policyFor(timeStampAuthority, releases), android.New(androidTrust),
	)
	if err != nil {
		return Verifier{reason: fmt.Errorf("make the statement verifier: %w", err)}
	}

	return Verifier{statementVerifier: statementVerifier}
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

func (v Verifier) VerdictOn(
	row yacymodel.URLMetadata,
	attestation attestedstatements.Bundle,
) Verdict {
	if v.reason != nil {
		return notAttested(v.reason)
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
	result, err := v.statementVerifier.Verify(bundle)
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
