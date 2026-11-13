package pageattestationcheck_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck"
	"github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck/pageattestationchecktest"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const otherSigningCertificate = "0f1e2d3c4b5a69788796a5b4c3d2e1f00f1e2d3c4b5a69788796a5b4c3d2e1f0"

func TestAnAppThatNoTrustedReleaseSignedIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, change := range map[string]func(*trustedappreleases.TrustedRelease){
		"other version code": func(r *trustedappreleases.TrustedRelease) {
			r.VersionCode = pageattestationchecktest.AppVersionCode + 1
		},
		"other signing certificate": func(r *trustedappreleases.TrustedRelease) {
			r.SigningCertificate = otherSigningCertificate
		},
	} {
		release := setup.trustedRelease()
		change(&release)

		verdict := setup.checkFor(t, release).VerdictOn(row, setup.attestationOf(t, row))

		if verdict.Attested {
			t.Errorf("%s: verdict = %+v, want not attested", name, verdict)
		}
	}
}

func TestAStatementOfAnotherAppIsNotAttested(t *testing.T) {
	setup := newSetup(t)
	setup.device = pageattestationchecktest.NewAttestingDeviceOfApp(
		t, "org.example.other", pageattestationchecktest.AppVersionCode,
	)
	row := rootedRow(t)

	verdict := setup.check(t).VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested {
		t.Errorf("verdict = %+v, want not attested", verdict)
	}
}

func TestAnAppReleaseIsAcceptedOnlyInStatementsStampedBeforeItsEndTime(t *testing.T) {
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

		verdict := setup.checkFor(t, release).VerdictOn(row, setup.attestationOf(t, row))

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

	verdict := setup.checkFor(t, other, setup.trustedRelease()).
		VerdictOn(row, setup.attestationOf(t, row))

	if !verdict.Attested {
		t.Errorf("verdict = %+v, want attested", verdict)
	}
}

func TestWithoutTrustedReleasesNoRowIsAttested(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)

	verdict := setup.checkFor(t).VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested || !errors.Is(verdict.Reason, pageattestationcheck.ErrNoTrustedReleases) {
		t.Errorf("verdict = %+v, want not attested for %v",
			verdict, pageattestationcheck.ErrNoTrustedReleases)
	}
}

func TestTrustedReleasesAttestRowsForTwoDaysAfterTheirRefresh(t *testing.T) {
	setup := newSetup(t)
	row := rootedRow(t)
	for name, test := range map[string]struct {
		age    time.Duration
		reason error
	}{
		"47 hours old": {47 * time.Hour, nil},
		"49 hours old": {49 * time.Hour, pageattestationcheck.ErrStaleTrustedReleases},
	} {
		check := setup.checkAged(t, test.age, setup.trustedRelease())

		verdict := check.VerdictOn(row, setup.attestationOf(t, row))

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

	verdict := setup.checkFor(t, setup.trustedRelease(), other).
		VerdictOn(row, setup.attestationOf(t, row))

	if verdict.Attested || verdict.Reason == nil {
		t.Errorf("verdict = %+v, want not attested with a reason", verdict)
	}
}
