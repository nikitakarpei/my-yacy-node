package pageattestationcheck

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"
	"github.com/nikitakarpei/hardware-attested-statements/verifier/profiles/android"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const policyFormat = "hardware-attested-statement/policy/v0.1"

var (
	ErrNoPageRWIRoot      = errors.New("the row has no page RWI root")
	ErrOtherPageRWIRoot   = errors.New("the attestation is of another page RWI root")
	ErrOtherAddress       = errors.New("the attestation is of another address")
	ErrHashOfOtherAddress = errors.New("the URL hash of the row is not the hash of its address")
	ErrVersionNotAccepted = errors.New("the policy does not accept the app version at that time")
	errNoAndroidSigner    = errors.New("the signer is no Android app")
)

type Check struct {
	verifier         *verifier.Verifier
	acceptedVersions map[int64]time.Time
}

type Verdict struct {
	Attested  bool
	FetchedAt time.Time
	Reason    error
}

func New(policy Policy, trust *android.Trust) (Check, error) {
	hasPolicy, err := verifier.ParsePolicy(hasPolicyOf(policy))
	if err != nil {
		return Check{}, fmt.Errorf("page attestation policy: %w", err)
	}
	statementVerifier, err := verifier.New(hasPolicy, android.New(trust))
	if err != nil {
		return Check{}, fmt.Errorf("page attestation verifier: %w", err)
	}

	return Check{
		verifier:         statementVerifier,
		acceptedVersions: endsOf(policy.AcceptedVersions),
	}, nil
}

func hasPolicyOf(policy Policy) []byte {
	versionCodes := make([]int64, 0, len(policy.AcceptedVersions))
	for _, version := range policy.AcceptedVersions {
		versionCodes = append(versionCodes, version.VersionCode)
	}
	encoding, _ := json.Marshal(
		map[string]any{ //nolint:errchkjson // strings, numbers and raw JSON.
			"policy":               policyFormat,
			"timestampAuthorities": policy.TimestampAuthorities,
			"drandChains":          []string{"quicknet"},
			"predicateTypes": map[string]any{
				pagerwicontract.PageRWIPredicateType: map[string]any{
					"trustedSigners": []map[string]any{{
						"profile":             android.Name,
						"package":             policy.Package,
						"signingCertificates": policy.SigningCertificates,
						"versionCodes":        versionCodes,
					}},
					"requireTimestamp":    true,
					"requireStartedAfter": true,
				},
			},
		},
	)

	return encoding
}

func endsOf(versions []AcceptedVersion) map[int64]time.Time {
	ends := make(map[int64]time.Time, len(versions))
	for _, version := range versions {
		ends[version.VersionCode] = version.Until
	}

	return ends
}

func (c Check) VerdictOn(metadata yacymodel.URLMetadata, attestation json.RawMessage) Verdict {
	bundle, err := verifier.ParseBundle(attestation)
	if err != nil {
		return notAttested(fmt.Errorf("read the attestation: %w", err))
	}
	statement, err := pagerwicontract.UnmarshalPageRWIStatement(bundle.DSSEEnvelope.Payload)
	if err != nil {
		return notAttested(err)
	}
	if err := failureOfRow(metadata, statement); err != nil {
		return notAttested(err)
	}
	result, err := c.verifier.Verify(bundle)
	if err != nil {
		return notAttested(fmt.Errorf("verify the attestation: %w", err))
	}
	if err := c.failureOfVersion(result); err != nil {
		return notAttested(err)
	}

	return Verdict{Attested: true, FetchedAt: *result.StartedAfter}
}

func notAttested(reason error) Verdict {
	return Verdict{Reason: reason}
}

func failureOfRow(
	metadata yacymodel.URLMetadata,
	statement pagerwicontract.PageRWIStatement,
) error {
	root, rooted := yacymodel.PageRWIRootFrom(metadata).Get()
	address, err := url.Parse(metadata.Address)
	switch {
	case !rooted:
		return ErrNoPageRWIRoot
	case root != statement.PageRWIRoot:
		return ErrOtherPageRWIRoot
	case statement.PageURL.String() != metadata.Address:
		return ErrOtherAddress
	case err != nil || yacymodel.URLHashOf(address) != metadata.Hash:
		return ErrHashOfOtherAddress
	default:
		return nil
	}
}

func (c Check) failureOfVersion(result *verifier.Result) error {
	app, isApp := result.Key.Signer.(*android.App)
	if !isApp {
		return errNoAndroidSigner
	}
	until := c.acceptedVersions[app.VersionCode]
	if !until.IsZero() && !result.NotAfter.Before(until) {
		return fmt.Errorf("%w: version %d until %v", ErrVersionNotAccepted, app.VersionCode, until)
	}

	return nil
}
