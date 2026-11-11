package pageattestationcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const policyFormat = "hardware-attested-statement/policy/v0.1"

var (
	ErrNoPageRWIRoot      = errors.New("the row has no page RWI root")
	ErrOtherPageRWIRoot   = errors.New("the attestation is of another page RWI root")
	ErrOtherAddress       = errors.New("the attestation is of another address")
	ErrHashOfOtherAddress = errors.New("the URL hash of the row is not the hash of its address")
)

type Check struct {
	verifier *verifier.Verifier
}

type Verdict struct {
	Attested  bool
	FetchedAt time.Time
	Reason    error
}

func New(policy Policy, trust *android.Trust) (Check, error) {
	statementPolicy, err := verifier.ParsePolicy(statementPolicyFrom(policy))
	if err != nil {
		return Check{}, fmt.Errorf("page attestation policy: %w", err)
	}
	statementVerifier, err := verifier.New(statementPolicy, android.New(trust))
	if err != nil {
		return Check{}, fmt.Errorf("page attestation verifier: %w", err)
	}

	return Check{verifier: statementVerifier}, nil
}

func statementPolicyFrom(policy Policy) []byte {
	versionCodes := make([]int64, 0, len(policy.AcceptedVersions))
	versionEndTimes := []map[string]int64{}
	for _, version := range policy.AcceptedVersions {
		versionCodes = append(versionCodes, version.VersionCode)
		if !version.EndTime.IsZero() {
			versionEndTimes = append(versionEndTimes, map[string]int64{
				"versionCode": version.VersionCode,
				"endTime":     version.EndTime.Unix(),
			})
		}
	}
	signer := map[string]any{
		"profile":             android.Name,
		"package":             policy.Package,
		"signingCertificates": policy.SigningCertificates,
		"versionCodes":        versionCodes,
	}
	if len(versionEndTimes) > 0 {
		signer["versionEndTimes"] = versionEndTimes
	}
	encoding, _ := json.Marshal(
		map[string]any{ //nolint:errchkjson // strings, numbers and raw JSON.
			"policy":               policyFormat,
			"timestampAuthorities": policy.TimeStampAuthorities,
			"drandChains":          []string{"quicknet"},
			"predicateTypes": map[string]any{
				pagerwicontract.PageRWIPredicateType: map[string]any{
					"trustedSigners":      []map[string]any{signer},
					"requireTimestamp":    true,
					"requireStartedAfter": true,
				},
			},
		},
	)

	return encoding
}

func (c Check) VerdictOn(
	row yacymodel.URLMetadata,
	attestation attestedstatements.Bundle,
) Verdict {
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
	result, err := c.verifier.Verify(bundle)
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
