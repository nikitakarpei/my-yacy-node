package github_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases"
	"github.com/nikitakarpei/yacy-rwi-node/trustedappreleases/releasesources/github"
)

const (
	repositoryID = "1273925787"
	tag          = "app-v1.0.0"
	commit       = "ada4d0e13a718d97ee957b8516a5f685d9816061"
	tagObject    = "5b3ec2f3bd4457d769f7ee6dc8591841e3099c67"
	record       = `{"signingCertificate":"aa","versionCode":1}`
	signedBundle = `{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`
)

type repository struct {
	answers map[string]string
	server  *httptest.Server
}

func newRepository(t *testing.T) *repository {
	t.Helper()
	repo := &repository{answers: map[string]string{}}
	repo.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer, found := repo.answers[r.URL.RequestURI()]
		if !found {
			http.NotFound(w, r)

			return
		}
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(repo.server.Close)
	repo.answers["/repositories/"+repositoryID+"/releases/tags/"+tag] = fmt.Sprintf(
		`{"tag_name":%q,"immutable":true,"assets":[`+
			`{"name":"release.json","browser_download_url":"%[2]s/download/release.json"},`+
			`{"name":"release.sigstore.json","browser_download_url":"%[2]s/download/release.sigstore.json"}]}`,
		tag, repo.server.URL)
	repo.answers["/download/release.json"] = record
	repo.answers["/download/release.sigstore.json"] = signedBundle
	repo.pointTagTo("commit", commit)

	return repo
}

func (r *repository) pointTagTo(objectType, sha string) {
	r.answers["/repositories/"+repositoryID+"/git/ref/tags/"+tag] = fmt.Sprintf(
		`{"object":{"type":%q,"sha":%q}}`, objectType, sha)
}

func (r *repository) source() github.Source {
	return github.New(r.server.Client(), r.server.URL, repositoryID)
}

func TestTheTagsAreThoseOfTheListedReleases(t *testing.T) {
	repo := newRepository(t)
	repo.answers["/repositories/"+repositoryID+"/releases?per_page=100"] = `[{"tag_name":"app-v1.0.1"},{"tag_name":"app-v1.0.0"}]`

	tags, err := repo.source().Tags(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if want := []string{"app-v1.0.1", "app-v1.0.0"}; !slices.Equal(tags, want) {
		t.Errorf("tags = %v, want %v", tags, want)
	}
}

func TestAReleaseHasTheCommitOfItsTagAndItsAssets(t *testing.T) {
	release, err := newRepository(t).source().ReleaseOf(t.Context(), tag)
	if err != nil {
		t.Fatal(err)
	}

	assertRelease(t, release)
}

func TestAReleaseOfAnAnnotatedTagHasTheCommitOfTheTagObject(t *testing.T) {
	repo := newRepository(t)
	repo.pointTagTo("tag", tagObject)
	repo.answers["/repositories/"+repositoryID+"/git/tags/"+tagObject] =
		`{"object":{"type":"commit","sha":"` + commit + `"}}`

	release, err := repo.source().ReleaseOf(t.Context(), tag)
	if err != nil {
		t.Fatal(err)
	}

	assertRelease(t, release)
}

func TestATagOfATreeGivesNoRelease(t *testing.T) {
	repo := newRepository(t)
	repo.pointTagTo("tree", tagObject)

	_, err := repo.source().ReleaseOf(t.Context(), tag)

	if !errors.Is(err, github.ErrTagWithoutCommit) {
		t.Errorf("error = %v, want %v", err, github.ErrTagWithoutCommit)
	}
}

func TestAReleaseWithoutItsRecordGivesNoRelease(t *testing.T) {
	repo := newRepository(t)
	key := "/repositories/" + repositoryID + "/releases/tags/" + tag
	repo.answers[key] = strings.Replace(repo.answers[key], `"release.json"`, `"other.json"`, 1)

	_, err := repo.source().ReleaseOf(t.Context(), tag)

	if !errors.Is(err, github.ErrMissingAsset) {
		t.Errorf("error = %v, want %v", err, github.ErrMissingAsset)
	}
}

func TestAnAnswerThatIsNotFoundGivesNoTags(t *testing.T) {
	_, err := newRepository(t).source().Tags(t.Context())

	if !errors.Is(err, github.ErrUnexpectedStatus) {
		t.Errorf("error = %v, want %v", err, github.ErrUnexpectedStatus)
	}
}

func TestAnAnswerThatIsTooLargeGivesNoRelease(t *testing.T) {
	repo := newRepository(t)
	repo.answers["/download/release.sigstore.json"] = strings.Repeat("a", 4<<20+1)

	_, err := repo.source().ReleaseOf(t.Context(), tag)

	if !errors.Is(err, github.ErrTooLarge) {
		t.Errorf("error = %v, want %v", err, github.ErrTooLarge)
	}
}

func TestAnAnswerThatIsNotJSONGivesNoTags(t *testing.T) {
	repo := newRepository(t)
	repo.answers["/repositories/"+repositoryID+"/releases?per_page=100"] = "<html>"

	if _, err := repo.source().Tags(t.Context()); err == nil {
		t.Error("an answer that is not JSON gave tags")
	}
}

func TestAnUnreachableGitHubGivesNoTags(t *testing.T) {
	repo := newRepository(t)
	repo.server.Close()

	if _, err := repo.source().Tags(t.Context()); err == nil {
		t.Error("an unreachable GitHub gave tags")
	}
}

func assertRelease(t *testing.T, release trustedappreleases.Release) {
	t.Helper()
	if release.Tag != tag || release.Commit != commit || !release.Immutable ||
		!bytes.Equal(
			release.Record,
			[]byte(record),
		) || !bytes.Equal(release.SignedBundle, []byte(signedBundle)) {
		t.Errorf("release = %+v", release)
	}
}
