package pageattestationcheck_test

import (
	"encoding/json"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/hardware-attested-statements/verifier"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck/pageattestationchecktest"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
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

	verdict := setup.check(t, setup.acceptedVersions()).VerdictOn(row, setup.attestationOf(t, row))

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

		verdict := setup.check(t, setup.acceptedVersions()).VerdictOn(row, attestation)

		if verdict.Attested || !errors.Is(verdict.Reason, change.reason) {
			t.Errorf("%s: verdict = %+v, want not attested for %v", name, verdict, change.reason)
		}
	}
}

func TestAnAttestationOfAnotherAddressIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	attestation := setup.attestationFor(t, row, "https://example.org/other")

	verdict := setup.check(t, setup.acceptedVersions()).VerdictOn(row, attestation)

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestationcheck.ErrOtherAddress) {
		t.Errorf("verdict = %+v", verdict)
	}
}

func TestARowWhoseHashIsNotOfItsAddressIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	other, _ := url.Parse("https://example.org/other")
	row.Hash = yacymodel.URLHashOf(other)

	verdict := setup.check(t, setup.acceptedVersions()).VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestationcheck.ErrHashOfOtherAddress) {
		t.Errorf("verdict = %+v", verdict)
	}
}

func TestAnAttestationThatDoesNotVerifyIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, attestation := range map[string]json.RawMessage{
		"no bundle":            json.RawMessage(`{"no":"bundle"}`),
		"no stamp":             setup.unstampedAttestationOf(t, row),
		"other statement type": setup.attestationOfStatement(t, []byte(`{"_type":"https://in-toto.io/Statement/v1"}`)),
	} {
		verdict := setup.check(t, setup.acceptedVersions()).VerdictOn(row, attestation)

		if verdict.Attested || verdict.Reason == nil {
			t.Errorf("%s: verdict = %+v, want not attested with a reason", name, verdict)
		}
	}
}

func TestAnAppVersionThePolicyDoesNotListIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	versions := []pageattestationcheck.AcceptedVersion{
		{VersionCode: pageattestationchecktest.AppVersionCode + 1},
	}

	verdict := setup.check(t, versions).VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
}

func TestAnAppVersionIsAcceptedOnlyInStatementsStampedBeforeItsEndTime(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, test := range map[string]struct {
		until    time.Time
		attested bool
	}{
		"ends later":   {setup.stampedAt.Add(time.Hour), true},
		"ended before": {setup.stampedAt.Add(-time.Hour), false},
	} {
		versions := []pageattestationcheck.AcceptedVersion{
			{VersionCode: pageattestationchecktest.AppVersionCode, Until: test.until},
		}

		verdict := setup.check(t, versions).VerdictOn(row, setup.attestationOf(t, row))

		if verdict.Attested != test.attested {
			t.Errorf("%s: verdict = %+v, want attested %v", name, verdict, test.attested)
		}
		if !test.attested &&
			!errors.Is(verdict.Reason, pageattestationcheck.ErrVersionNotAccepted) {
			t.Errorf("%s: reason = %v", name, verdict.Reason)
		}
	}
}

func TestNewRefusesAPolicyWithoutATimestampAuthority(t *testing.T) {
	setup := newSetup(t)
	policy := setup.policy(setup.acceptedVersions())
	policy.TimestampAuthorities = []json.RawMessage{json.RawMessage(`"unknown"`)}

	if _, err := pageattestationcheck.New(policy, setup.device.Trust()); err == nil {
		t.Error("a policy with an unknown time-stamp authority is accepted")
	}
}

type setup struct {
	device    *pageattestationchecktest.AttestingDevice
	authority *pageattestationchecktest.TimestampAuthority
	stampedAt time.Time
}

func newSetup(t *testing.T) setup {
	t.Helper()

	return setup{
		device: pageattestationchecktest.NewAttestingDevice(
			t,
			pageattestationchecktest.AppVersionCode,
		),
		authority: pageattestationchecktest.NewTimestampAuthority(t),
		stampedAt: time.Now().Add(-time.Minute).Truncate(time.Second),
	}
}

func (s setup) acceptedVersions() []pageattestationcheck.AcceptedVersion {
	return []pageattestationcheck.AcceptedVersion{
		{VersionCode: pageattestationchecktest.AppVersionCode},
	}
}

func (s setup) policy(versions []pageattestationcheck.AcceptedVersion) pageattestationcheck.Policy {
	return pageattestationcheck.Policy{
		Package:              pageattestationchecktest.AppPackage,
		SigningCertificates:  []string{pageattestationchecktest.AppSigner},
		AcceptedVersions:     versions,
		TimestampAuthorities: []json.RawMessage{s.authority.PolicyEntry()},
	}
}

func (s setup) check(
	t *testing.T,
	versions []pageattestationcheck.AcceptedVersion,
) pageattestationcheck.Check {
	t.Helper()
	check, err := pageattestationcheck.New(s.policy(versions), s.device.Trust())
	if err != nil {
		t.Fatal(err)
	}

	return check
}

func (s setup) attestationOf(t *testing.T, row yacymodel.URLMetadata) json.RawMessage {
	t.Helper()

	return s.attestationFor(t, row, row.Address)
}

func (s setup) attestationFor(
	t *testing.T,
	row yacymodel.URLMetadata,
	address string,
) json.RawMessage {
	t.Helper()

	return s.attestationOfStatement(t, statementOf(t, row, address))
}

func (s setup) attestationOfStatement(t *testing.T, statement []byte) json.RawMessage {
	t.Helper()
	bundle := s.device.BundleOf(t, statement)
	s.authority.Stamp(t, bundle, s.stampedAt)

	return encodingOf(t, bundle)
}

func (s setup) unstampedAttestationOf(t *testing.T, row yacymodel.URLMetadata) json.RawMessage {
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

func encodingOf(t *testing.T, bundle *verifier.Bundle) json.RawMessage {
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
