package trustedappreleases_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
)

var (
	refreshTime     = time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	errUnreachable  = errors.New("unreachable")
	errOtherSubject = errors.New("the bundle attests another subject")
)

type publishedReleases struct {
	listingErr error
	releases   map[string]trustedappreleases.Release
	unreadable map[string]bool
	readings   map[string]int
	tags       []string
}

func newPublishedReleases() *publishedReleases {
	return &publishedReleases{
		releases:   map[string]trustedappreleases.Release{},
		unreadable: map[string]bool{},
		readings:   map[string]int{},
	}
}

func (p *publishedReleases) publish(release trustedappreleases.Release) {
	p.tags = append(p.tags, release.Tag)
	p.releases[release.Tag] = release
}

func (p *publishedReleases) TagsOldestFirst(context.Context) ([]string, error) {
	return p.tags, p.listingErr
}

func (p *publishedReleases) ReleaseOf(
	_ context.Context,
	tag string,
) (trustedappreleases.Release, error) {
	p.readings[tag]++
	if p.unreadable[tag] {
		return trustedappreleases.Release{}, errUnreachable
	}

	return p.releases[tag], nil
}

type fakeDenials struct {
	err     error
	denials []trustedappreleases.Denial
}

func (d *fakeDenials) Denials(context.Context) ([]trustedappreleases.Denial, error) {
	return d.denials, d.err
}

type refreshEvents struct {
	events []string
}

func (r *refreshEvents) Refreshed(_ context.Context, amountOfTrustedReleases int) {
	r.events = append(r.events, fmt.Sprintf("refreshed %d", amountOfTrustedReleases))
}

func (r *refreshEvents) DenialsUnread(context.Context, error) {
	r.events = append(r.events, "denials unread")
}

func (r *refreshEvents) ReleasesUnlisted(context.Context, error) {
	r.events = append(r.events, "releases unlisted")
}

func (r *refreshEvents) ReleaseUnread(_ context.Context, tag string, _ error) {
	r.events = append(r.events, "unread "+tag)
}

func (r *refreshEvents) ReleaseUntrusted(_ context.Context, tag string, _ error) {
	r.events = append(r.events, "untrusted "+tag)
}

func (r *refreshEvents) CacheUnread(context.Context, error) {
	r.events = append(r.events, "cache unread")
}

func (r *refreshEvents) CacheUnwritten(context.Context, error) {
	r.events = append(r.events, "cache unwritten")
}

type attestedRecords map[[sha256.Size]byte]trustedappreleases.Provenance

func (a attestedRecords) ProvenanceOf(
	_ []byte,
	subject [sha256.Size]byte,
) (trustedappreleases.Provenance, error) {
	provenance, attested := a[subject]
	if !attested {
		return trustedappreleases.Provenance{}, errOtherSubject
	}

	return provenance, nil
}

type refreshSetup struct {
	published *publishedReleases
	denied    *fakeDenials
	observed  *refreshEvents
	attested  attestedRecords
	cachePath string
}

func newRefreshSetup(t *testing.T) *refreshSetup {
	t.Helper()

	return &refreshSetup{
		published: newPublishedReleases(),
		denied:    &fakeDenials{},
		observed:  &refreshEvents{},
		attested:  attestedRecords{},
		cachePath: filepath.Join(t.TempDir(), "trusted-releases.json"),
	}
}

func (s *refreshSetup) open(t *testing.T) *trustedappreleases.TrustedReleases {
	t.Helper()

	return trustedappreleases.New(
		trustedappreleases.OpenReleaseCache(t.Context(), s.cachePath, s.observed),
		workflow,
		s.attested,
		s.published,
		s.denied,
		s.observed,
	)
}

func (s *refreshSetup) publishAttested(tag, certificate string) trustedappreleases.TrustedRelease {
	versionCode := int64(len(s.published.tags) + 1)
	releaseRecord := fmt.Appendf(
		nil,
		`{"signingCertificate":%q,"versionCode":%d}`,
		certificate,
		versionCode,
	)
	s.published.publish(
		trustedappreleases.Release{
			Tag:             tag,
			Commit:          commit,
			Record:          releaseRecord,
			Immutable:       true,
			OnReleaseBranch: true,
		},
	)
	s.attested[sha256.Sum256(releaseRecord)] = trustedappreleases.Provenance{
		WorkflowAddress:   repository + "/" + workflow.Path + "@refs/tags/" + tag,
		RepositoryAddress: repository,
		RepositoryID:      workflow.RepositoryID,
		OwnerID:           workflow.OwnerID,
		Ref:               "refs/tags/" + tag,
		Commit:            commit,
		Runner:            "github-hosted",
		Visibility:        "public",
	}

	return trustedappreleases.TrustedRelease{
		Tag:                tag,
		SigningCertificate: certificate,
		VersionCode:        versionCode,
	}
}

func certificateNumbered(number int) string {
	return fmt.Sprintf("%064x", number)
}

func assertCurrent(
	t *testing.T,
	releases *trustedappreleases.TrustedReleases,
	denialsReadAt time.Time,
	want ...trustedappreleases.TrustedRelease,
) {
	t.Helper()
	current, currentDenialsReadAt := releases.Releases(), releases.DenialsReadAt()
	if !slices.Equal(current, want) || !currentDenialsReadAt.Equal(denialsReadAt) {
		t.Errorf(
			"current = %+v with denials read at %v, want %+v with denials read at %v",
			current,
			currentDenialsReadAt,
			want,
			denialsReadAt,
		)
	}
}

func TestARefreshTrustsTheAttestedReleasesOfTheWorkflow(t *testing.T) {
	setup := newRefreshSetup(t)
	first := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	second := setup.publishAttested("app-v1.1.0", certificateNumbered(2))
	releases := setup.open(t)

	releases.Refresh(t.Context(), refreshTime)

	assertCurrent(t, releases, refreshTime, first, second)
	if want := []string{"refreshed 2"}; !slices.Equal(setup.observed.events, want) {
		t.Errorf("events = %v, want %v", setup.observed.events, want)
	}
}

func TestARefreshReadsOnlyNewReleasesOfTheWorkflow(t *testing.T) {
	setup := newRefreshSetup(t)
	setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	setup.published.tags = append(setup.published.tags, "other-v1.0.0")
	releases := setup.open(t)
	releases.Refresh(t.Context(), refreshTime)

	releases.Refresh(t.Context(), refreshTime.Add(time.Hour))

	if setup.published.readings["app-v1.0.0"] != 1 ||
		setup.published.readings["other-v1.0.0"] != 0 {
		t.Errorf("readings = %v, want one of the release of the workflow", setup.published.readings)
	}
}

func TestAReleaseThatIsNoLongerListedStaysTrusted(t *testing.T) {
	setup := newRefreshSetup(t)
	trusted := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	releases := setup.open(t)
	releases.Refresh(t.Context(), refreshTime)
	setup.published.tags = nil

	releases.Refresh(t.Context(), refreshTime.Add(time.Hour))

	assertCurrent(t, releases, refreshTime.Add(time.Hour), trusted)
}

func TestTheEarliestDenialOfACertificateDeniesItsReleases(t *testing.T) {
	setup := newRefreshSetup(t)
	denied := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	other := setup.publishAttested("app-v1.1.0", certificateNumbered(2))
	setup.denied.denials = []trustedappreleases.Denial{
		{SigningCertificate: denied.SigningCertificate, DeniedFrom: refreshTime.Add(-time.Hour)},
		{
			SigningCertificate: denied.SigningCertificate,
			DeniedFrom:         refreshTime.Add(-2 * time.Hour),
		},
	}
	releases := setup.open(t)

	releases.Refresh(t.Context(), refreshTime)

	denied.DeniedFrom = refreshTime.Add(-2 * time.Hour)
	assertCurrent(t, releases, refreshTime, denied, other)
}

func TestARemovedDenialNoLongerDeniesTheRelease(t *testing.T) {
	setup := newRefreshSetup(t)
	trusted := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	setup.denied.denials = []trustedappreleases.Denial{
		{SigningCertificate: trusted.SigningCertificate, DeniedFrom: refreshTime},
	}
	releases := setup.open(t)
	releases.Refresh(t.Context(), refreshTime)
	setup.denied.denials = nil

	releases.Refresh(t.Context(), refreshTime.Add(time.Hour))

	assertCurrent(t, releases, refreshTime.Add(time.Hour), trusted)
}

func TestUnreadDenialsKeepTheLastDenialsAndTheirReadTime(t *testing.T) {
	setup := newRefreshSetup(t)
	first := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	setup.denied.denials = []trustedappreleases.Denial{
		{SigningCertificate: first.SigningCertificate, DeniedFrom: refreshTime},
	}
	releases := setup.open(t)
	releases.Refresh(t.Context(), refreshTime)
	setup.denied.err = errUnreachable
	second := setup.publishAttested("app-v1.1.0", certificateNumbered(2))

	releases.Refresh(t.Context(), refreshTime.Add(time.Hour))

	first.DeniedFrom = refreshTime
	assertCurrent(t, releases, refreshTime, first, second)
	if !slices.Contains(setup.observed.events, "denials unread") {
		t.Errorf("events = %v, want the denials unread", setup.observed.events)
	}
}

func TestDenialsApplyWhileTheReleasesAreUnlisted(t *testing.T) {
	setup := newRefreshSetup(t)
	trusted := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	releases := setup.open(t)
	releases.Refresh(t.Context(), refreshTime)
	setup.published.listingErr = errUnreachable
	setup.denied.denials = []trustedappreleases.Denial{
		{SigningCertificate: trusted.SigningCertificate, DeniedFrom: refreshTime},
	}

	releases.Refresh(t.Context(), refreshTime.Add(time.Hour))

	trusted.DeniedFrom = refreshTime
	assertCurrent(t, releases, refreshTime.Add(time.Hour), trusted)
	if !slices.Contains(setup.observed.events, "releases unlisted") {
		t.Errorf("events = %v, want the releases unlisted", setup.observed.events)
	}
}

func TestAReleaseThatIsUnreadOrUntrustedIsNotTrustedAndIsReadAgain(t *testing.T) {
	setup := newRefreshSetup(t)
	trusted := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	setup.publishAttested("app-v1.1.0", certificateNumbered(2))
	setup.published.unreadable["app-v1.1.0"] = true
	setup.published.publish(
		trustedappreleases.Release{
			Tag:             "app-v1.2.0",
			Commit:          commit,
			Record:          []byte(record),
			Immutable:       true,
			OnReleaseBranch: true,
		},
	)
	releases := setup.open(t)
	releases.Refresh(t.Context(), refreshTime)

	releases.Refresh(t.Context(), refreshTime.Add(time.Hour))

	assertCurrent(t, releases, refreshTime.Add(time.Hour), trusted)
	for _, event := range []string{"unread app-v1.1.0", "untrusted app-v1.2.0", "refreshed 1"} {
		if !slices.Contains(setup.observed.events, event) {
			t.Errorf("events = %v, want %q", setup.observed.events, event)
		}
	}
	if setup.published.readings["app-v1.1.0"] != 2 || setup.published.readings["app-v1.2.0"] != 2 {
		t.Errorf("readings = %v, want two of each release not trusted", setup.published.readings)
	}
}

func TestReopenedReleasesAreThoseOfTheLastRefresh(t *testing.T) {
	setup := newRefreshSetup(t)
	trusted := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	setup.denied.denials = []trustedappreleases.Denial{
		{SigningCertificate: trusted.SigningCertificate, DeniedFrom: refreshTime},
	}
	setup.open(t).Refresh(t.Context(), refreshTime)
	setup.published.tags = nil

	reopened := setup.open(t)

	trusted.DeniedFrom = refreshTime
	assertCurrent(t, reopened, refreshTime, trusted)
}

func TestAReopenedReleaseIsNotReadAgain(t *testing.T) {
	setup := newRefreshSetup(t)
	setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	setup.open(t).Refresh(t.Context(), refreshTime)
	reopened := setup.open(t)

	reopened.Refresh(t.Context(), refreshTime.Add(time.Hour))

	if readings := setup.published.readings["app-v1.0.0"]; readings != 1 {
		t.Errorf("readings = %d, want the one before the reopening", readings)
	}
}

func TestAnUnreadableCacheGivesNoTrustedReleases(t *testing.T) {
	setup := newRefreshSetup(t)
	if err := os.WriteFile(setup.cachePath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	releases := setup.open(t)

	assertCurrent(t, releases, time.Time{})
	if want := []string{"cache unread"}; !slices.Equal(setup.observed.events, want) {
		t.Errorf("events = %v, want %v", setup.observed.events, want)
	}
}

func TestAnUnwritableCacheKeepsTheRefreshInMemory(t *testing.T) {
	setup := newRefreshSetup(t)
	setup.cachePath = filepath.Join(t.TempDir(), "missing", "trusted-releases.json")
	trusted := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	releases := setup.open(t)

	releases.Refresh(t.Context(), refreshTime)

	assertCurrent(t, releases, refreshTime, trusted)
	if !slices.ContainsFunc(setup.observed.events, func(event string) bool {
		return strings.HasPrefix(event, "cache unwritten")
	}) {
		t.Errorf("events = %v, want the cache unwritten", setup.observed.events)
	}
}

func TestAReleaseThatReusesTheCertificateOfATrustedReleaseIsNotTrusted(t *testing.T) {
	setup := newRefreshSetup(t)
	trusted := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	releases := setup.open(t)
	releases.Refresh(t.Context(), refreshTime)
	setup.publishAttested("app-v1.1.0", certificateNumbered(1))

	releases.Refresh(t.Context(), refreshTime.Add(time.Hour))

	assertCurrent(t, releases, refreshTime.Add(time.Hour), trusted)
	if !slices.Contains(setup.observed.events, "untrusted app-v1.1.0") {
		t.Errorf(
			"events = %v, want the release that reuses a certificate untrusted",
			setup.observed.events,
		)
	}
}

func TestOfTwoNewReleasesWithOneCertificateOnlyTheOlderIsTrusted(t *testing.T) {
	setup := newRefreshSetup(t)
	older := setup.publishAttested("app-v1.0.0", certificateNumbered(1))
	setup.publishAttested("app-v1.1.0", certificateNumbered(1))
	releases := setup.open(t)

	releases.Refresh(t.Context(), refreshTime)

	assertCurrent(t, releases, refreshTime, older)
}
