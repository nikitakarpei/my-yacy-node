package pageattestations_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pageattestations"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestations/pageattestationstest"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const otherSigningCertificate = "0f1e2d3c4b5a69788796a5b4c3d2e1f00f1e2d3c4b5a69788796a5b4c3d2e1f0"

func TestARowSignedByNoTrustedReleaseIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, change := range map[string]func(*trustedappreleases.TrustedRelease){
		"other version code": func(r *trustedappreleases.TrustedRelease) {
			r.VersionCode = pageattestationstest.AppVersionCode + 1
		},
		"other signing certificate": func(r *trustedappreleases.TrustedRelease) {
			r.SigningCertificate = otherSigningCertificate
		},
	} {
		release := setup.trustedRelease()
		change(&release)

		verdict := setup.verifierFor(t, release).VerdictOn(row, setup.attestationOf(t, row))

		if verdict.Attested {
			t.Errorf("%s: verdict = %+v, want not attested", name, verdict)
		}
	}
}

func TestAStatementOfAnotherAppIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	setup.device = pageattestationstest.NewAttestingDeviceOfApp(
		t, "org.example.other", pageattestationstest.AppVersionCode,
	)
	row := rootedRow(t)

	verdict := setup.verifier(t).VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
}

func TestATrustedReleaseAttestsOnlyStatementsStampedBeforeItsEndTime(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, test := range map[string]struct {
		endTime  time.Time
		attested bool
	}{
		"ends later":   {setup.stampedAt.Add(time.Hour), true},
		"ended before": {setup.stampedAt.Add(-time.Hour), false},
	} {
		release := setup.trustedRelease()
		release.EndTime = test.endTime

		verdict := setup.verifierFor(t, release).VerdictOn(row, setup.attestationOf(t, row))

		if verdict.Attested != test.attested {
			t.Errorf("%s: verdict = %+v, want attested %v", name, verdict, test.attested)
		}
	}
}

func TestAnyOfTheTrustedReleasesAttestsARow(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	other := setup.trustedRelease()
	other.SigningCertificate = otherSigningCertificate

	verdict := setup.verifierFor(t, other, setup.trustedRelease()).
		VerdictOn(row, setup.attestationOf(t, row))

	if !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
}

func TestWithoutTrustedReleasesNoRowIsAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)

	verdict := setup.verifierFor(t).VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestations.ErrNoTrustedReleases) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestations.ErrNoTrustedReleases)
	}
}

func TestTrustedReleasesAttestRowsForTwoDaysAfterTheDenialsAreRead(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, test := range map[string]struct {
		age    time.Duration
		reason error
	}{
		"47 hours old": {47 * time.Hour, nil},
		"49 hours old": {49 * time.Hour, pageattestations.ErrStaleReleaseDenials},
	} {
		attestationVerifier := setup.verifierWithDenialsReadBefore(
			t,
			test.age,
			setup.trustedRelease(),
		)

		verdict := attestationVerifier.VerdictOn(row, setup.attestationOf(t, row))

		if verdict.Attested != (test.reason == nil) || !errors.Is(verdict.Reason, test.reason) {
			t.Errorf("%s: verdict = %+v, want reason %v", name, verdict, test.reason)
		}
	}
}

func TestTrustedReleasesThatShareASigningCertificateAttestNoRow(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	other := setup.trustedRelease()
	other.Tag = "yacypeercompanion-v1.0.1"

	verdict := setup.verifierFor(t, setup.trustedRelease(), other).
		VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested || verdict.Reason == nil {
		t.Errorf("verdict = %+v, want not attested with a reason", verdict)
	}
}
