package pageattestationcheck

import (
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

var (
	ErrNoPageRWIRoot      = errors.New("the row has no page RWI root")
	ErrOtherPageRWIRoot   = errors.New("the attestation is of another page RWI root")
	ErrOtherAddress       = errors.New("the attestation is of another address")
	ErrHashOfOtherAddress = errors.New(
		"the URL hash of the row is not the hash of its address",
	)
	ErrPolicyWithoutFetchTimeBounds = errors.New(
		"the policy does not require a time-stamp and a drand round of page RWI statements",
	)
)

type Check struct {
	verifier *verifier.Verifier
}

type Verdict struct {
	Attested  bool
	FetchedAt time.Time
	Reason    error
}

func New(policy []byte, trust *android.Trust) (Check, error) {
	statementPolicy, err := verifier.ParsePolicy(policy)
	if err != nil {
		return Check{}, fmt.Errorf("page attestation policy: %w", err)
	}
	if !requiresFetchTimeBounds(statementPolicy) {
		return Check{}, ErrPolicyWithoutFetchTimeBounds
	}
	statementVerifier, err := verifier.New(statementPolicy, android.New(trust))
	if err != nil {
		return Check{}, fmt.Errorf("page attestation verifier: %w", err)
	}

	return Check{verifier: statementVerifier}, nil
}

func requiresFetchTimeBounds(policy *verifier.Policy) bool {
	pageRWIPolicy, listed := policy.PredicateTypes[pagerwicontract.PageRWIPredicateType]

	return listed && pageRWIPolicy.RequireTimestamp && pageRWIPolicy.RequireStartedAfter
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
