package trustedappreleases

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"
)

var ErrReusedSigningCertificate = errors.New(
	"the signing certificate of the release is that of another trusted release",
)

type Sources struct {
	Releases ReleaseSource
	Denials  []DenialSource
}

type TrustedReleases struct {
	source        ReleaseSource
	observer      RefreshObserver
	verified      map[string]TrustedRelease
	refreshedAt   time.Time
	cachePath     string
	denialSources []DenialSource
	denials       []Denial
	check         Check
}

type cachedReleases struct {
	RefreshedAt time.Time        `json:"refreshedAt"`
	Releases    []TrustedRelease `json:"releases"`
	Denials     []Denial         `json:"denials"`
}

func Open(
	ctx context.Context,
	cachePath string,
	check Check,
	sources Sources,
	observer RefreshObserver,
) *TrustedReleases {
	releases := &TrustedReleases{
		source:        sources.Releases,
		observer:      observer,
		verified:      map[string]TrustedRelease{},
		cachePath:     cachePath,
		denialSources: sources.Denials,
		check:         check,
	}
	releases.restore(ctx)

	return releases
}

func (r *TrustedReleases) restore(ctx context.Context) {
	content, err := os.ReadFile(r.cachePath)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	var cached cachedReleases
	if err == nil {
		err = json.Unmarshal(content, &cached)
	}
	if err != nil {
		r.observer.CacheUnread(ctx, fmt.Errorf("read the cache of trusted releases: %w", err))

		return
	}
	for _, release := range cached.Releases {
		r.verified[release.Tag] = release
	}
	r.denials = cached.Denials
	r.refreshedAt = cached.RefreshedAt
}

func (r *TrustedReleases) Refresh(ctx context.Context, now time.Time) {
	denials, read := r.currentDenials(ctx)
	if !read {
		return
	}
	tags, err := r.source.TagsOldestFirst(ctx)
	if err != nil {
		r.observer.ReleasesUnlisted(ctx, err)

		return
	}
	for _, tag := range r.newTags(tags) {
		r.verify(ctx, tag)
	}
	r.denials = denials
	r.refreshedAt = now
	r.keep(ctx)
	r.observer.Refreshed(ctx, len(r.verified))
}

func (r *TrustedReleases) currentDenials(ctx context.Context) ([]Denial, bool) {
	var denials []Denial
	for _, source := range r.denialSources {
		sourceDenials, err := source.Denials(ctx)
		if err != nil {
			r.observer.DenialsUnread(ctx, err)

			return nil, false
		}
		denials = append(denials, sourceDenials...)
	}

	return denials, true
}

func (r *TrustedReleases) newTags(tags []string) []string {
	return slices.DeleteFunc(slices.Clone(tags), func(tag string) bool {
		_, verified := r.verified[tag]

		return verified || !r.check.Covers(tag)
	})
}

func (r *TrustedReleases) verify(ctx context.Context, tag string) {
	release, err := r.source.ReleaseOf(ctx, tag)
	if err != nil {
		r.observer.ReleaseUnread(ctx, tag, err)

		return
	}
	verdict := r.check.VerdictOn(release)
	if verdict.Trusted && r.trustsCertificate(verdict.Release.SigningCertificate) {
		verdict = Verdict{Reason: ErrReusedSigningCertificate}
	}
	if !verdict.Trusted {
		r.observer.ReleaseUntrusted(ctx, tag, verdict.Reason)

		return
	}
	r.verified[tag] = verdict.Release
}

func (r *TrustedReleases) trustsCertificate(signingCertificate string) bool {
	for _, release := range r.verified {
		if release.SigningCertificate == signingCertificate {
			return true
		}
	}

	return false
}

func (r *TrustedReleases) keep(ctx context.Context) {
	content, err := json.Marshal(cachedReleases{
		RefreshedAt: r.refreshedAt,
		Releases:    slices.SortedFunc(maps.Values(r.verified), byTag),
		Denials:     r.denials,
	})
	if err == nil {
		err = writeReplacing(r.cachePath, content)
	}
	if err != nil {
		r.observer.CacheUnwritten(ctx, err)
	}
}

func writeReplacing(path string, content []byte) error {
	written, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write the cache of trusted releases: %w", err)
	}
	defer func() { _ = os.Remove(written.Name()) }()
	_, err = written.Write(content)
	if closeErr := written.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(written.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("write the cache of trusted releases: %w", err)
	}

	return nil
}

func (r *TrustedReleases) Current() ([]TrustedRelease, time.Time) {
	releases := make([]TrustedRelease, 0, len(r.verified))
	for _, release := range r.verified {
		release.EndTime = endTimeOf(release.SigningCertificate, r.denials)
		releases = append(releases, release)
	}
	slices.SortFunc(releases, byTag)

	return releases, r.refreshedAt
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
