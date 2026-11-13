package trustedappreleases_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

const (
	tag                = "app-v1.0.0"
	commit             = "ada4d0e13a718d97ee957b8516a5f685d9816061"
	signingCertificate = "72e21ab017f94f81137bb0d983453d02cfb59b31eac7935c31473a3ad2b2c428"
	record             = `{"signingCertificate":"` + signingCertificate + `","versionCode":7}`
	repository         = "https://github.com/owner/app"
)

var (
	workflow = trustedappreleases.ReleaseWorkflow{
		RepositoryID: "11",
		OwnerID:      "22",
		Path:         ".github/workflows/release.yml",
		TagPattern:   "app-v*",
	}
	release = trustedappreleases.Release{
		Tag:       tag,
		Commit:    commit,
		Record:    []byte(record),
		Immutable: true,
	}
)

func TestAnImmutableReleaseOfTheWorkflowAtItsTagIsTrusted(t *testing.T) {
	verdict := workflow.VerdictOn(release, provenanceOfRelease())

	want := trustedappreleases.TrustedRelease{
		Tag:                tag,
		SigningCertificate: signingCertificate,
		VersionCode:        7,
	}
	if !verdict.Trusted || verdict.Reason != nil || verdict.TrustedRelease != want {
		t.Errorf("verdict = %+v, want trusted %+v", verdict, want)
	}
}

func TestAReleaseOutsideTheWorkflowIsNotTrusted(t *testing.T) {
	for name, change := range map[string]struct {
		release func(*trustedappreleases.Release)
		reason  error
	}{
		"mutable":       {func(r *trustedappreleases.Release) { r.Immutable = false }, trustedappreleases.ErrMutableRelease},
		"other tag":     {func(r *trustedappreleases.Release) { r.Tag = "other-v1.0.0" }, trustedappreleases.ErrTagOutsidePattern},
		"malformed tag": {func(r *trustedappreleases.Release) { r.Tag = "[" }, trustedappreleases.ErrTagOutsidePattern},
	} {
		changed := release
		change.release(&changed)

		verdict := workflow.VerdictOn(changed, provenanceOfRelease())

		if verdict.Trusted || !errors.Is(verdict.Reason, change.reason) {
			t.Errorf("%s: verdict = %+v, want not trusted for %v", name, verdict, change.reason)
		}
	}
}

func TestAProvenanceOfAnotherRunIsNotTrusted(t *testing.T) {
	for name, change := range map[string]struct {
		provenance func(*trustedappreleases.Provenance)
		reason     error
	}{
		"other repository": {func(p *trustedappreleases.Provenance) { p.RepositoryID = "12" }, trustedappreleases.ErrOtherRepository},
		"other owner":      {func(p *trustedappreleases.Provenance) { p.OwnerID = "23" }, trustedappreleases.ErrOtherOwner},
		"other ref":        {func(p *trustedappreleases.Provenance) { p.Ref = "refs/heads/main" }, trustedappreleases.ErrOtherRef},
		"other workflow": {func(p *trustedappreleases.Provenance) {
			p.WorkflowAddress = repository + "/.github/workflows/other.yml@refs/tags/" + tag
		}, trustedappreleases.ErrOtherWorkflow},
		"workflow of another repository": {func(p *trustedappreleases.Provenance) {
			p.WorkflowAddress = "https://github.com/other/app/" + workflow.Path + "@refs/tags/" + tag
		}, trustedappreleases.ErrOtherWorkflow},
		"other commit":       {func(p *trustedappreleases.Provenance) { p.Commit = "0" + commit[1:] }, trustedappreleases.ErrOtherCommit},
		"self-hosted":        {func(p *trustedappreleases.Provenance) { p.Runner = "self-hosted" }, trustedappreleases.ErrSelfHostedRunner},
		"private repository": {func(p *trustedappreleases.Provenance) { p.Visibility = "private" }, trustedappreleases.ErrPrivateRepository},
	} {
		provenance := provenanceOfRelease()
		change.provenance(&provenance)

		verdict := workflow.VerdictOn(release, provenance)

		if verdict.Trusted || !errors.Is(verdict.Reason, change.reason) {
			t.Errorf("%s: verdict = %+v, want not trusted for %v", name, verdict, change.reason)
		}
	}
}

func TestAMalformedRecordIsNotTrusted(t *testing.T) {
	for name, malformed := range map[string]string{
		"not JSON":             "{",
		"short certificate":    `{"signingCertificate":"72e2","versionCode":7}`,
		"upper case":           `{"signingCertificate":"72E21AB017F94F81137BB0D983453D02CFB59B31EAC7935C31473A3AD2B2C428","versionCode":7}`,
		"no versionCode":       `{"signingCertificate":"` + signingCertificate + `"}`,
		"negative versionCode": `{"signingCertificate":"` + signingCertificate + `","versionCode":-1}`,
	} {
		malformedRelease := release
		malformedRelease.Record = []byte(malformed)

		verdict := workflow.VerdictOn(malformedRelease, provenanceOfRelease())

		if verdict.Trusted || !errors.Is(verdict.Reason, trustedappreleases.ErrMalformedRecord) {
			t.Errorf("%s: verdict = %+v, want not trusted for a malformed record", name, verdict)
		}
	}
}

func provenanceOfRelease() trustedappreleases.Provenance {
	return trustedappreleases.Provenance{
		WorkflowAddress:   repository + "/" + workflow.Path + "@refs/tags/" + tag,
		RepositoryAddress: repository,
		RepositoryID:      workflow.RepositoryID,
		OwnerID:           workflow.OwnerID,
		Ref:               "refs/tags/" + tag,
		Commit:            commit,
		Runner:            "github-hosted",
		Visibility:        "public",
	}
}
