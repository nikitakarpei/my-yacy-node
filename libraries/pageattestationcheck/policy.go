package pageattestationcheck

import (
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const maxTrustedReleasesAge = 48 * time.Hour

var (
	ErrPolicyWithPredicateTypes = errors.New(
		"the policy names predicate types, which the check sets from the trusted app releases",
	)
	ErrNoTrustedReleases    = errors.New("no app release is trusted")
	ErrStaleTrustedReleases = errors.New(
		"the trusted app releases are older than their maximum age",
	)
)

type Policy struct {
	policy     verifier.Policy
	appPackage string
}

func PolicyFrom(encoding []byte, appPackage string) (Policy, error) {
	policy, err := verifier.ParsePolicy(encoding)
	if err != nil {
		return Policy{}, fmt.Errorf("page attestation policy: %w", err)
	}
	if len(policy.PredicateTypes) > 0 {
		return Policy{}, ErrPolicyWithPredicateTypes
	}
	if _, err := verifier.New(policy); err != nil {
		return Policy{}, fmt.Errorf("page attestation policy: %w", err)
	}

	return Policy{policy: *policy, appPackage: appPackage}, nil
}

func (p Policy) CheckFor(
	releases []trustedappreleases.TrustedRelease,
	refreshedAt time.Time,
	androidTrust *android.Trust,
	now time.Time,
) Check {
	switch {
	case len(releases) == 0:
		return Check{reason: ErrNoTrustedReleases}
	case now.Sub(refreshedAt) > maxTrustedReleasesAge:
		return Check{reason: fmt.Errorf(
			"%w: refreshed at %s", ErrStaleTrustedReleases, refreshedAt.Format(time.RFC3339),
		)}
	}
	statementVerifier, err := verifier.New(p.policyFor(releases), android.New(androidTrust))
	if err != nil {
		return Check{reason: fmt.Errorf("page attestation verifier: %w", err)}
	}

	return Check{verifier: statementVerifier}
}

func (p Policy) policyFor(releases []trustedappreleases.TrustedRelease) *verifier.Policy {
	policy := p.policy
	policy.PredicateTypes = map[string]verifier.PredicatePolicy{
		pagerwicontract.PageRWIPredicateType: {
			TrustedSigners:      p.signersFor(releases),
			RequireTimestamp:    true,
			RequireStartedAfter: true,
		},
	}

	return &policy
}

func (p Policy) signersFor(releases []trustedappreleases.TrustedRelease) []jsontext.Value {
	signers := make([]jsontext.Value, 0, len(releases))
	for _, release := range releases {
		signer := android.Signer{
			Profile:             android.Name,
			Package:             p.appPackage,
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
