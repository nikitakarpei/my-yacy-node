package trustedappreleases

import (
	"cmp"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"
)

var ErrReusedSigningCertificate = errors.New(
	"the signing certificate of the release is that of another trusted release",
)

type TrustedReleases struct {
	trusted            map[string]TrustedRelease
	denialsReadAt      time.Time
	denials            []Denial
	cache              *ReleaseCache
	workflow           ReleaseWorkflow
	provenanceVerifier ProvenanceVerifier
	releaseSource      ReleaseSource
	denialSource       DenialSource
	observer           RefreshObserver
}

func New( //nolint:revive // argument-limit: each argument is one collaborator of the refresh.
	cache *ReleaseCache,
	workflow ReleaseWorkflow,
	provenanceVerifier ProvenanceVerifier,
	releaseSource ReleaseSource,
	denialSource DenialSource,
	observer RefreshObserver,
) *TrustedReleases {
	lastRefresh := cache.LastRefresh()
	trusted := map[string]TrustedRelease{}
	for _, release := range lastRefresh.Releases {
		trusted[release.Tag] = release
	}

	return &TrustedReleases{
		trusted:            trusted,
		denialsReadAt:      lastRefresh.DenialsReadAt,
		denials:            lastRefresh.Denials,
		cache:              cache,
		workflow:           workflow,
		provenanceVerifier: provenanceVerifier,
		releaseSource:      releaseSource,
		denialSource:       denialSource,
		observer:           observer,
	}
}

func (r *TrustedReleases) Refresh(ctx context.Context, now time.Time) {
	r.readDenials(ctx, now)
	r.trustListedReleases(ctx)
	r.cache.Write(ctx, r.cachedRefresh())
	r.observer.Refreshed(ctx, len(r.trusted))
}

func (r *TrustedReleases) readDenials(ctx context.Context, now time.Time) {
	denials, err := r.denialSource.Denials(ctx)
	if err != nil {
		r.observer.DenialsUnread(ctx, err)

		return
	}
	r.denials = denials
	r.denialsReadAt = now
}

func (r *TrustedReleases) trustListedReleases(ctx context.Context) {
	tags, err := r.releaseSource.TagsOldestFirst(ctx)
	if err != nil {
		r.observer.ReleasesUnlisted(ctx, err)

		return
	}
	for _, tag := range r.untrustedTagsFrom(tags) {
		r.trust(ctx, tag)
	}
}

func (r *TrustedReleases) untrustedTagsFrom(tags []string) []string {
	return slices.DeleteFunc(slices.Clone(tags), func(tag string) bool {
		_, trusted := r.trusted[tag]

		return trusted || !r.workflow.Covers(tag)
	})
}

func (r *TrustedReleases) trust(ctx context.Context, tag string) {
	release, err := r.releaseSource.ReleaseOf(ctx, tag)
	if err != nil {
		r.observer.ReleaseUnread(ctx, tag, err)

		return
	}
	verdict := r.verdictOn(release)
	if !verdict.Trusted {
		r.observer.ReleaseUntrusted(ctx, tag, verdict.Reason)

		return
	}
	r.trusted[tag] = verdict.TrustedRelease
}

func (r *TrustedReleases) verdictOn(release Release) Verdict {
	provenance, err := r.provenanceVerifier.ProvenanceOf(
		release.SignedBundle, sha256.Sum256(release.Record),
	)
	if err != nil {
		return untrusted(fmt.Errorf("verify the provenance of the release record: %w", err))
	}
	verdict := r.workflow.VerdictOn(release, provenance)
	if verdict.Trusted && r.trustsSigningCertificate(verdict.TrustedRelease.SigningCertificate) {
		return untrusted(ErrReusedSigningCertificate)
	}

	return verdict
}

func (r *TrustedReleases) trustsSigningCertificate(signingCertificate string) bool {
	for _, release := range r.trusted {
		if release.SigningCertificate == signingCertificate {
			return true
		}
	}

	return false
}

func (r *TrustedReleases) cachedRefresh() CachedRefresh {
	return CachedRefresh{
		DenialsReadAt: r.denialsReadAt,
		Releases:      slices.SortedFunc(maps.Values(r.trusted), byTag),
		Denials:       r.denials,
	}
}

func (r *TrustedReleases) Releases() []TrustedRelease {
	releases := make([]TrustedRelease, 0, len(r.trusted))
	for _, release := range r.trusted {
		release.EndTime = endTimeOf(release.SigningCertificate, r.denials)
		releases = append(releases, release)
	}
	slices.SortFunc(releases, byTag)

	return releases
}

func byTag(a, b TrustedRelease) int {
	return cmp.Compare(a.Tag, b.Tag)
}

func endTimeOf(signingCertificate string, denials []Denial) time.Time {
	var endTime time.Time
	for _, denial := range denials {
		if denial.SigningCertificate == signingCertificate &&
			(endTime.IsZero() || denial.EndTime.Before(endTime)) {
			endTime = denial.EndTime
		}
	}

	return endTime
}

func (r *TrustedReleases) DenialsReadAt() time.Time {
	return r.denialsReadAt
}
