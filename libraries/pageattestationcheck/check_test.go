package pageattestationcheck_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"

	"github.com/nikitakarpei/yacy-rwi-node/attestedstatements"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck/pageattestationchecktest"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	pageAddress    = "https://example.org/a"
	quicknetRound  = 33667957
	roundSignature = "ace44ad88121ed26163c00acaa4c89522295c4fb066a8e9712138c63644e43cadac17e3f29aa8056f8f4ef5449d6e16e"
)

func TestARowWithItsPageAttestationIsAttestedAtTheTimeOfItsRound(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)

	verdict := setup.check(t).VerdictOn(row, setup.attestationOf(t, row))

	if !verdict.Attested || verdict.Reason != nil {
		t.Fatalf("verdict = %+v, want attested", verdict)
	}
	if want := quicknet(t).RoundTime(quicknetRound); !verdict.FetchedAt.Equal(want) {
		t.Errorf("fetched at = %v, want the round time %v", verdict.FetchedAt, want)
	}
}

func TestARowThatDiffersFromItsAttestationIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	for name, change := range map[string]struct {
		row    func(*yacymodel.URLMetadata)
		reason error
	}{
		"changed title": {func(r *yacymodel.URLMetadata) { r.Title = "Other" }, pageattestationcheck.ErrOtherPageRWIRoot},
		"no root":       {func(r *yacymodel.URLMetadata) { r.PostingsRoot = yacymodel.None[yacymodel.PostingsRoot]() }, pageattestationcheck.ErrNoPageRWIRoot},
		"other address": {func(r *yacymodel.URLMetadata) { r.Address = "https://example.org/b" }, pageattestationcheck.ErrOtherPageRWIRoot},
	} {
		row := rootedRow(t)
		attestation := setup.attestationOf(t, row)
		change.row(&row)

		verdict := setup.check(t).VerdictOn(row, attestation)

		if verdict.Attested || !errors.Is(verdict.Reason, change.reason) {
			t.Errorf("%s: verdict = %+v, want not attested for %v", name, verdict, change.reason)
		}
	}
}

func TestAnAttestationOfAnotherAddressIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	attestation := setup.attestationFor(t, row, "https://example.org/other")

	verdict := setup.check(t).VerdictOn(row, attestation)

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestationcheck.ErrOtherAddress) {
		t.Errorf("verdict = %+v", verdict)
	}
}

func TestARowWhoseHashIsNotOfItsAddressIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	other, _ := url.Parse("https://example.org/other")
	row.Hash = yacymodel.URLHashOf(other)

	verdict := setup.check(t).VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestationcheck.ErrHashOfOtherAddress) {
		t.Errorf("verdict = %+v", verdict)
	}
}

func TestAnAttestationThatDoesNotVerifyIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, attestation := range map[string]attestedstatements.Bundle{
		"no bundle":            attestedstatements.Bundle(`{"no":"bundle"}`),
		"no stamp":             setup.unstampedAttestationOf(t, row),
		"other statement type": setup.attestationOfStatement(t, []byte(`{"_type":"https://in-toto.io/Statement/v1"}`)),
	} {
		verdict := setup.check(t).VerdictOn(row, attestation)

		if verdict.Attested || verdict.Reason == nil {
			t.Errorf("%s: verdict = %+v, want not attested with a reason", name, verdict)
		}
	}
}

type setup struct {
	device    *pageattestationchecktest.AttestingDevice
	authority *pageattestationchecktest.TimeStampAuthority
	stampedAt time.Time
}

func newSetup(t *testing.T) setup {
	t.Helper()

	return setup{
		device: pageattestationchecktest.NewAttestingDevice(
			t,
			pageattestationchecktest.AppVersionCode,
		),
		authority: pageattestationchecktest.NewTimeStampAuthority(t),
		stampedAt: time.Now().Add(-time.Minute).Truncate(time.Second),
	}
}

func (s setup) trustedRelease() trustedappreleases.TrustedRelease {
	return trustedappreleases.TrustedRelease{
		Tag:                "yacypeercompanion-v1.0.0",
		SigningCertificate: pageattestationchecktest.AppSigningCertificate,
		VersionCode:        pageattestationchecktest.AppVersionCode,
	}
}

func (s setup) policy(t *testing.T) pageattestationcheck.Policy {
	t.Helper()
	policy, err := pageattestationcheck.PolicyFrom(
		pageattestationchecktest.PolicyFor(s.authority),
		pageattestationchecktest.AppPackage,
	)
	if err != nil {
		t.Fatal(err)
	}

	return policy
}

func (s setup) check(t *testing.T) pageattestationcheck.Check {
	t.Helper()

	return s.checkFor(t, s.trustedRelease())
}

func (s setup) checkFor(
	t *testing.T,
	releases ...trustedappreleases.TrustedRelease,
) pageattestationcheck.Check {
	t.Helper()
	now := time.Now()

	return s.policy(t).CheckFor(releases, now, s.device.Trust(), now)
}

func (s setup) attestationOf(t *testing.T, row yacymodel.URLMetadata) attestedstatements.Bundle {
	t.Helper()

	return s.attestationFor(t, row, row.Address)
}

func (s setup) attestationFor(
	t *testing.T,
	row yacymodel.URLMetadata,
	address string,
) attestedstatements.Bundle {
	t.Helper()

	return s.attestationOfStatement(t, statementOf(t, row, address))
}

func (s setup) attestationOfStatement(t *testing.T, statement []byte) attestedstatements.Bundle {
	t.Helper()
	bundle := s.device.BundleOf(t, statement)
	s.authority.Stamp(t, bundle, s.stampedAt)

	return encodingOf(t, bundle)
}

func (s setup) unstampedAttestationOf(
	t *testing.T,
	row yacymodel.URLMetadata,
) attestedstatements.Bundle {
	t.Helper()

	return encodingOf(t, s.device.BundleOf(t, statementOf(t, row, row.Address)))
}

func statementOf(t *testing.T, row yacymodel.URLMetadata, address string) []byte {
	t.Helper()
	root, _ := yacymodel.PageRWIRootFrom(row).Get()
	pageURL, err := canonicalurl.CanonicalURLOf(address)
	if err != nil {
		t.Fatal(err)
	}
	startedAfter := json.RawMessage(`{"drand":{"chain":"` + quicknet(t).Hash +
		`","round":33667957,"signature":"` + roundSignature + `"}}`)

	return pagerwicontract.MarshalPageRWIStatement(
		pagerwicontract.PageRWIStatementOf(root, pageURL, pageURL, startedAfter),
	)
}

func encodingOf(t *testing.T, bundle *verifier.Bundle) attestedstatements.Bundle {
	t.Helper()
	encoding, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}

	return encoding
}

func rootedRow(t *testing.T) yacymodel.URLMetadata {
	t.Helper()
	address, err := url.Parse(pageAddress)
	if err != nil {
		t.Fatal(err)
	}
	urlHash := yacymodel.URLHashOf(address)
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(
		[]yacymodel.RWIPosting{{WordHash: yacymodel.WordHash("page"), URLHash: urlHash}},
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		t.Fatal(err)
	}

	return yacymodel.URLMetadata{
		Hash:         urlHash,
		Address:      pageAddress,
		Title:        "A page",
		PostingsRoot: pagePostings.Root(),
	}
}

func quicknet(t *testing.T) verifier.DrandChain {
	t.Helper()
	chain, err := verifier.DrandQuicknet()
	if err != nil {
		t.Fatal(err)
	}

	return chain
}
