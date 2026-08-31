package searchendpoint_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	defaultSearchResultCount = 10
	defaultSearchTimeLimit   = 3 * time.Second
	maxSearchTimeLimit       = 3 * time.Second
)

var searchWord = searchtest.HashFor("w1")

type searchDocument struct {
	Address    string
	Language   string
	Appearance yacymodel.Appearance
}

func mountedSearchFor(
	t *testing.T,
	word yacymodel.Hash,
	documents ...searchDocument,
) *http.ServeMux {
	t.Helper()

	index, directory := searchFixtureFor(t, word, documents...)

	return mountedSearch(t, index, directory)
}

func searchFixtureFor(
	t *testing.T,
	word yacymodel.Hash,
	documents ...searchDocument,
) (searchtest.PostingIndex, searchtest.URLDirectory) {
	t.Helper()

	postings := make([]yacymodel.RWIPosting, 0, len(documents))
	stored := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(documents))
	for _, document := range documents {
		hash := urlHashOf(t, document.Address)
		postings = append(postings, yacymodel.RWIPosting{
			WordHash:   word,
			URLHash:    hash,
			Hits:       1,
			Appearance: document.Appearance,
			Language:   languageIn(t, document),
		})
		stored[hash] = yacymodel.URLMetadata{Address: document.Address}
	}

	return searchtest.PostingIndex{
			Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{word: postings},
		},
		searchtest.URLDirectory{Documents: stored}
}

func urlHashOf(t *testing.T, address string) yacymodel.URLHash {
	t.Helper()

	return yacymodel.URLNormalformOf(parsedAddress(t, address)).Hash()
}

func parsedAddress(t *testing.T, address string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(address)
	if err != nil {
		t.Fatalf("parse %q: %v", address, err)
	}

	return parsed
}

func languageIn(t *testing.T, document searchDocument) yacymodel.Optional[yacymodel.Language] {
	t.Helper()

	if document.Language == "" {
		return yacymodel.None[yacymodel.Language]()
	}

	language, err := yacymodel.ParseLanguage(document.Language)
	if err != nil {
		t.Fatalf("ParseLanguage(%q): %v", document.Language, err)
	}

	return yacymodel.Some(language)
}

func searchRequestFor(
	word yacymodel.Hash,
	options yacyproto.SearchRequest,
) yacyproto.SearchRequest {
	options.NetworkName = searchNetwork
	options.Query = []yacymodel.Hash{word}

	return options
}

func assertDocuments(t *testing.T, resp yacyproto.SearchResponse, want ...string) {
	t.Helper()

	found := make([]string, len(resp.Resources))
	for position, resource := range resp.Resources {
		found[position] = resource.Address
	}
	slices.Sort(found)

	expected := slices.Clone(want)
	slices.Sort(expected)

	if !slices.Equal(found, expected) {
		t.Errorf("documents = %v, want %v", found, expected)
	}
}

const (
	chosenSite  = "http://example.com/a"
	ignoredSite = "http://other.example/b"
)

func TestSiteFilterFollowsTheFieldPrecedence(t *testing.T) {
	mux := mountedSearchFor(t, searchWord,
		searchDocument{Address: chosenSite},
		searchDocument{Address: ignoredSite},
	)

	for _, testCase := range []struct {
		name    string
		options yacyproto.SearchRequest
	}{
		{"site hash before the operator and the host", yacyproto.SearchRequest{
			SiteHash: hostHashOf(t, chosenSite).String(),
			Modifier: "site:other.example",
			SiteHost: "other.example",
		}},
		{"operator before the host", yacyproto.SearchRequest{
			Modifier: "site:example.com",
			SiteHost: "other.example",
		}},
		{"host alone", yacyproto.SearchRequest{SiteHost: "example.com"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resp := search(t, mux, searchRequestFor(searchWord, testCase.options))

			assertDocuments(t, resp, chosenSite)
		})
	}
}

func hostHashOf(t *testing.T, address string) yacymodel.HostHash {
	t.Helper()

	return yacymodel.URLNormalformOf(parsedAddress(t, address)).HostHash()
}

func TestSiteHostIsNormalizedBeforeItIsHashed(t *testing.T) {
	mux := mountedSearchFor(t, searchWord,
		searchDocument{Address: chosenSite},
		searchDocument{Address: ignoredSite},
	)

	for _, site := range []string{"example.com", "Example.COM", ".example.com.", "EXAMPLE.com"} {
		t.Run(site, func(t *testing.T) {
			resp := search(t, mux, searchRequestFor(
				searchWord,
				yacyproto.SearchRequest{SiteHost: site},
			))

			assertDocuments(t, resp, chosenSite)
		})
	}
}

func TestSiteHostReadsAnFtpPrefixedHostAsAnFtpSite(t *testing.T) {
	overFTP, overHTTP := "ftp://ftp.example.com/a", "http://ftp.example.com/a"
	mux := mountedSearchFor(t, searchWord,
		searchDocument{Address: overFTP},
		searchDocument{Address: overHTTP},
	)

	resp := search(t, mux, searchRequestFor(
		searchWord,
		yacyproto.SearchRequest{SiteHost: "ftp.example.com"},
	))

	assertDocuments(t, resp, overFTP)
}

func TestSiteHostThatNamesNoHostIsRejected(t *testing.T) {
	mux := mountedSearchFor(t, searchWord)

	for _, site := range []string{".", "..."} {
		t.Run(site, func(t *testing.T) {
			rec := postSearch(t, mux, searchRequestFor(
				searchWord,
				yacyproto.SearchRequest{SiteHost: site},
			))

			if rec.Code == http.StatusOK {
				t.Fatalf("search accepted site host %q, which names no host", site)
			}
		})
	}
}

func TestOnlyTheLanguageOperatorFiltersByLanguage(t *testing.T) {
	german, english := "http://example.com/de", "http://example.com/en"
	mux := mountedSearchFor(t, searchWord,
		searchDocument{Address: german, Language: "de"},
		searchDocument{Address: english, Language: "en"},
	)

	for _, testCase := range []struct {
		name    string
		options yacyproto.SearchRequest
		want    []string
	}{
		{
			"the operator filters and outranks the structured field",
			yacyproto.SearchRequest{Modifier: "/language/de", Language: "en"},
			[]string{german},
		},
		{
			"the structured field alone does not filter",
			yacyproto.SearchRequest{Language: "en"},
			[]string{german, english},
		},
		{
			"an operator of the wrong length does not filter",
			yacyproto.SearchRequest{Modifier: "/language/deu"},
			[]string{german, english},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			resp := search(t, mux, searchRequestFor(searchWord, testCase.options))

			assertDocuments(t, resp, testCase.want...)
		})
	}
}

func TestContentDomainFiltersByTheAppearanceItNames(t *testing.T) {
	matching, plain := "http://example.com/match", "http://example.com/plain"

	for _, testCase := range []struct {
		name       string
		domain     yacyproto.SearchContentDomain
		appearance yacymodel.Appearance
		want       []string
	}{
		{
			"image", yacyproto.ContentDomainImage,
			yacymodel.Appearance{HasImage: true},
			[]string{matching},
		},
		{
			"audio", yacyproto.ContentDomainAudio,
			yacymodel.Appearance{HasAudio: true},
			[]string{matching},
		},
		{
			"video", yacyproto.ContentDomainVideo,
			yacymodel.Appearance{HasVideo: true},
			[]string{matching},
		},
		{
			"app", yacyproto.ContentDomainApp,
			yacymodel.Appearance{HasApp: true},
			[]string{matching},
		},
		{
			"text", yacyproto.ContentDomainText,
			yacymodel.Appearance{HasImage: true},
			[]string{matching, plain},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			mux := mountedSearchFor(t, searchWord,
				searchDocument{Address: matching, Appearance: testCase.appearance},
				searchDocument{Address: plain},
			)

			resp := search(t, mux, searchRequestFor(
				searchWord,
				yacyproto.SearchRequest{ContentDom: testCase.domain},
			))

			assertDocuments(t, resp, testCase.want...)
		})
	}
}

func TestMissingCountTakesTheDefaultResultCount(t *testing.T) {
	documents := make([]searchDocument, defaultSearchResultCount+5)
	for position := range documents {
		documents[position] = searchDocument{
			Address: fmt.Sprintf("http://example.com/u%02d", position),
		}
	}
	mux := mountedSearchFor(t, searchWord, documents...)

	resp := search(t, mux, searchRequestFor(searchWord, yacyproto.SearchRequest{}))

	if resp.Count != defaultSearchResultCount {
		t.Errorf("Count = %d, want the default of %d", resp.Count, defaultSearchResultCount)
	}
}

func TestMissingTimeTakesTheDefaultSearchTimeLimit(t *testing.T) {
	remaining := remainingSearchTimeFor(t, yacyproto.SearchRequest{})

	assertRemainingSearchTime(t, remaining, defaultSearchTimeLimit)
}

func remainingSearchTimeFor(t *testing.T, options yacyproto.SearchRequest) time.Duration {
	t.Helper()

	index, directory := searchFixtureFor(t, searchWord,
		searchDocument{Address: chosenSite},
	)
	recording := &deadlineRecordingPostingIndex{postings: index}
	mux, _ := mountedSearchResults(
		t,
		searchresult.New(openVault(t), recording, directory, maxPostingsPerTerm),
	)

	search(t, mux, searchRequestFor(searchWord, options))

	if recording.remainingSearchTime == 0 {
		t.Fatal("the search ran without a deadline")
	}

	return recording.remainingSearchTime
}

type deadlineRecordingPostingIndex struct {
	postings            searchtest.PostingIndex
	remainingSearchTime time.Duration
}

func (index *deadlineRecordingPostingIndex) ScanWord(
	ctx context.Context,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	if deadline, found := ctx.Deadline(); found {
		index.remainingSearchTime = time.Until(deadline)
	}

	return index.postings.ScanWord(ctx, word, visit)
}

func (index *deadlineRecordingPostingIndex) RWICount(ctx context.Context) (int, error) {
	return index.postings.RWICount(ctx)
}

func (index *deadlineRecordingPostingIndex) PostingOf(
	ctx context.Context,
	word yacymodel.Hash,
	document yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	return index.postings.PostingOf(ctx, word, document)
}

func assertRemainingSearchTime(t *testing.T, remaining, want time.Duration) {
	t.Helper()

	const tolerance = 100 * time.Millisecond

	if remaining > want || remaining < want-tolerance {
		t.Errorf("remaining search time = %v, want close to %v", remaining, want)
	}
}

func TestExcessiveTimeIsClampedToTheLongestSearchTimeLimit(t *testing.T) {
	requested := 10 * time.Second

	remaining := remainingSearchTimeFor(t, yacyproto.SearchRequest{
		Time: int(requested / time.Millisecond),
	})

	if remaining >= requested {
		t.Errorf(
			"remaining search time = %v, want less than the requested %v",
			remaining,
			requested,
		)
	}
	assertRemainingSearchTime(t, remaining, maxSearchTimeLimit)
}

func TestTimeBelowTheLongestLimitIsKept(t *testing.T) {
	requested := 500 * time.Millisecond

	remaining := remainingSearchTimeFor(t, yacyproto.SearchRequest{
		Time: int(requested / time.Millisecond),
	})

	assertRemainingSearchTime(t, remaining, requested)
}
