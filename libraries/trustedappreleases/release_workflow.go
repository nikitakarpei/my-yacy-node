package trustedappreleases

import (
	"errors"
	"path"
)

const (
	githubHostedRunner = "github-hosted"
	publicVisibility   = "public"
	tagRefPrefix       = "refs/tags/"
)

var (
	ErrMutableRelease    = errors.New("the release is not immutable")
	ErrOffReleaseBranch  = errors.New("the commit of the tag is not on the release branch")
	ErrTagOutsidePattern = errors.New(
		"the tag does not match the tag pattern of the release workflow",
	)
	ErrOtherRepository   = errors.New("the provenance names another repository")
	ErrOtherOwner        = errors.New("the provenance names another owner")
	ErrOtherRef          = errors.New("the provenance names another ref than the tag")
	ErrOtherWorkflow     = errors.New("the provenance names another workflow")
	ErrOtherCommit       = errors.New("the provenance names another commit than the tag")
	ErrSelfHostedRunner  = errors.New("the provenance names a runner that GitHub does not host")
	ErrPrivateRepository = errors.New("the provenance names a repository that was not public")
)

type ReleaseWorkflow struct {
	RepositoryID string
	OwnerID      string
	Path         string
	TagPattern   string
}

type Verdict struct {
	Reason         error
	TrustedRelease TrustedRelease
	Trusted        bool
}

func (w ReleaseWorkflow) VerdictOn(release Release, provenance Provenance) Verdict {
	if reason := w.reasonAgainstRelease(release); reason != nil {
		return untrusted(reason)
	}
	if reason := w.reasonAgainstProvenance(provenance, release); reason != nil {
		return untrusted(reason)
	}
	trustedRelease, err := trustedReleaseFrom(release.Tag, release.Record)
	if err != nil {
		return untrusted(err)
	}

	return Verdict{Trusted: true, TrustedRelease: trustedRelease}
}

func untrusted(reason error) Verdict {
	return Verdict{Reason: reason}
}

func (w ReleaseWorkflow) reasonAgainstRelease(release Release) error {
	switch {
	case !release.Immutable:
		return ErrMutableRelease
	case !release.OnReleaseBranch:
		return ErrOffReleaseBranch
	case !w.Covers(release.Tag):
		return ErrTagOutsidePattern
	default:
		return nil
	}
}

func (w ReleaseWorkflow) Covers(tag string) bool {
	matched, err := path.Match(w.TagPattern, tag)

	return err == nil && matched
}

func (w ReleaseWorkflow) reasonAgainstProvenance(provenance Provenance, release Release) error {
	ref := tagRefPrefix + release.Tag
	switch {
	case provenance.RepositoryID != w.RepositoryID:
		return ErrOtherRepository
	case provenance.OwnerID != w.OwnerID:
		return ErrOtherOwner
	case provenance.Ref != ref:
		return ErrOtherRef
	case provenance.WorkflowAddress != provenance.RepositoryAddress+"/"+w.Path+"@"+ref:
		return ErrOtherWorkflow
	case provenance.Commit != release.Commit:
		return ErrOtherCommit
	case provenance.Runner != githubHostedRunner:
		return ErrSelfHostedRunner
	case provenance.Visibility != publicVisibility:
		return ErrPrivateRepository
	default:
		return nil
	}
}
