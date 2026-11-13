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
	ErrTagOutsidePattern = errors.New(
		"the tag does not match the tag pattern of the release workflow",
	)
	ErrOtherRepository   = errors.New("the provenance names another repository")
	ErrOtherOwner        = errors.New("the provenance names another owner")
	ErrOtherWorkflow     = errors.New("the provenance names another workflow or ref")
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

func (w ReleaseWorkflow) reasonAgainstRelease(release Release) error {
	matched, err := path.Match(w.TagPattern, release.Tag)
	switch {
	case !release.Immutable:
		return ErrMutableRelease
	case err != nil || !matched:
		return ErrTagOutsidePattern
	default:
		return nil
	}
}

func (w ReleaseWorkflow) reasonAgainstProvenance(provenance Provenance, release Release) error {
	ref := tagRefPrefix + release.Tag
	switch {
	case provenance.RepositoryID != w.RepositoryID:
		return ErrOtherRepository
	case provenance.OwnerID != w.OwnerID:
		return ErrOtherOwner
	case provenance.Ref != ref || provenance.Workflow != provenance.Repository+"/"+w.Path+"@"+ref:
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
