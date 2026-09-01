package searchresult_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/matchreport"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

func mustLanguage(t *testing.T, raw string) yacymodel.Optional[yacymodel.Language] {
	t.Helper()

	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		t.Fatalf("parse language %q: %v", raw, err)
	}

	return yacymodel.Some(language)
}

func postingEntry(word yacymodel.Hash, url string, position int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:     word,
		URLHash:      searchtest.URLHashFor(url),
		Hits:         1,
		TextPosition: position,
	}
}

func addressFor(id string) string {
	return "http://example.com/" + id
}

func urlMetadata(ids ...string) map[yacymodel.URLHash]yacymodel.URLMetadata {
	metadata := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(ids))
	for _, id := range ids {
		metadata[searchtest.URLHashFor(id)] = yacymodel.URLMetadata{Address: addressFor(id)}
	}

	return metadata
}

func hasExactlyDocuments(got []yacymodel.URLHash, ids ...string) bool {
	if len(got) != len(ids) {
		return false
	}
	want := make(map[yacymodel.URLHash]struct{}, len(ids))
	for _, id := range ids {
		want[searchtest.URLHashFor(id)] = struct{}{}
	}
	for _, hash := range got {
		if _, ok := want[hash]; !ok {
			return false
		}
	}

	return true
}

func TestSearchJoinsAndCountsAndReports(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "u1", 0), postingEntry(word1, "u2", 0)},
		word2: {postingEntry(word2, "u2", 0), postingEntry(word2, "u3", 0)},
	}}
	results := searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.URLDirectory{Documents: urlMetadata("u1", "u2", "u3")},
	)

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
		matchreport.RequestedReport{Mode: matchreport.TermWithMostMatches},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if result.TotalDocumentsMatchingEveryTerm != 1 {
		t.Errorf(
			"TotalDocumentsMatchingEveryTerm = %d, want 1",
			result.TotalDocumentsMatchingEveryTerm,
		)
	}
	if len(result.DocumentMetadata) != 1 {
		t.Fatalf("resources = %d, want 1", len(result.DocumentMetadata))
	}
	if result.DocumentMetadata[0].Address != addressFor("u2") {
		t.Errorf("resource = %v, want u2", result.DocumentMetadata[0])
	}
	if result.TotalMatchesPerTerm[word1] != 2 {
		t.Errorf("TotalMatchesPerTerm[w1] = %d, want 2", result.TotalMatchesPerTerm[word1])
	}
	if got := result.DocumentsMatchingEachReportedTerm[word1]; !hasExactlyDocuments(
		got,
		"u1",
		"u2",
	) {
		t.Errorf("DocumentsMatchingEachReportedTerm[w1] = %v, want u1, u2", got)
	}
}

func TestSearchTakesMostRelevantUpToLimit(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {
			postingEntry(word, "u1", 0),
			postingEntry(word, "u2", 0),
			postingEntry(word, "u3", 0),
		},
	}}
	results := searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.URLDirectory{Documents: urlMetadata("u1", "u2", "u3")},
	)

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}, MaxResults: 2},
		matchreport.RequestedReport{},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if result.TotalDocumentsMatchingEveryTerm != 3 {
		t.Errorf(
			"TotalDocumentsMatchingEveryTerm = %d, want 3",
			result.TotalDocumentsMatchingEveryTerm,
		)
	}
	if len(result.DocumentMetadata) != 2 {
		t.Errorf("resources = %d, want 2", len(result.DocumentMetadata))
	}
}

func TestSearchFiltersByAverageGapNotSpan(t *testing.T) {
	word1, word2, word3 := searchtest.HashFor(
		"w1",
	), searchtest.HashFor(
		"w2",
	), searchtest.HashFor(
		"w3",
	)
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "uA", 1), postingEntry(word1, "uB", 1)},
		word2: {postingEntry(word2, "uA", 5), postingEntry(word2, "uB", 10)},
		word3: {postingEntry(word3, "uA", 9), postingEntry(word3, "uB", 20)},
	}}
	results := searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.URLDirectory{Documents: urlMetadata("uA", "uB")},
	)

	result, err := results.ResultFor(context.Background(), searchcriteria.Criteria{
		Terms:         []yacymodel.Hash{word1, word2, word3},
		MaxTermSpread: 5,
	}, matchreport.RequestedReport{})
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if len(result.DocumentMetadata) != 1 ||
		result.DocumentMetadata[0].Address != addressFor("uA") {
		t.Fatalf("resources = %v, want only uA (span 8, average gap 4)", result.DocumentMetadata)
	}
}

func TestSearchSurfacesExcludedTermScanFailures(t *testing.T) {
	results := searchresult.New(
		openVault(t),
		termpostings.New(searchtest.FailingPostingIndex{Err: errScanBroken}, 100),
		searchtest.URLDirectory{},
	)

	_, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{ExcludedTerms: []yacymodel.Hash{searchtest.HashFor("ban")}},
		matchreport.RequestedReport{},
	)
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("ResultFor error = %v, want %v", err, errScanBroken)
	}
}

func TestSearchSurfacesQueryTermScanFailures(t *testing.T) {
	results := searchresult.New(
		openVault(t),
		termpostings.New(searchtest.FailingPostingIndex{Err: errScanBroken}, 100),
		searchtest.URLDirectory{},
	)

	_, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{searchtest.HashFor("w1")}},
		matchreport.RequestedReport{},
	)
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("ResultFor error = %v, want %v", err, errScanBroken)
	}
}

func TestSearchSurfacesMetadataFailures(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1", 0)},
	}}
	results := searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.FailingURLDirectory{Err: errDirectoryBroken},
	)

	_, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		matchreport.RequestedReport{},
	)
	if !errors.Is(err, errDirectoryBroken) {
		t.Fatalf("ResultFor error = %v, want %v", err, errDirectoryBroken)
	}
}

func TestSearchSurfacesReportedTermScanFailures(t *testing.T) {
	results := searchresult.New(
		openVault(t),
		termpostings.New(searchtest.FailingPostingIndex{Err: errScanBroken}, 100),
		searchtest.URLDirectory{},
	)

	_, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{},
		matchreport.RequestedReport{
			Mode:  matchreport.RequestedTerms,
			Terms: []yacymodel.Hash{searchtest.HashFor("w2")},
		},
	)
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("ResultFor error = %v, want %v", err, errScanBroken)
	}
}

var (
	errScanBroken      = errors.New("scan broken")
	errDirectoryBroken = errors.New("directory broken")
)

func TestSearchReportsRequestedTermsWithoutWantedTerms(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1", 1), postingEntry(word, "u2", 1)},
	}}
	results := searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.URLDirectory{},
	)

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{},
		matchreport.RequestedReport{
			Mode:  matchreport.RequestedTerms,
			Terms: []yacymodel.Hash{word},
		},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if result.TotalDocumentsMatchingEveryTerm != 0 || len(result.DocumentMetadata) != 0 {
		t.Fatalf("result = %+v, want report only", result)
	}
	if result.TotalMatchesPerTerm[word] != 2 {
		t.Errorf("TotalMatchesPerTerm = %d, want 2", result.TotalMatchesPerTerm[word])
	}
	if got := result.DocumentsMatchingEachReportedTerm[word]; !hasExactlyDocuments(
		got,
		"u1",
		"u2",
	) {
		t.Errorf("DocumentsMatchingEachReportedTerm = %v, want u1, u2", got)
	}
}

func TestSearchReportsRequestedTermsAlongsideWantedTerms(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word:    {postingEntry(word, "u1", 0), postingEntry(word, "u2", 0)},
		related: {postingEntry(related, "u2", 0), postingEntry(related, "u3", 0)},
	}}
	results := searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")},
	)

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		matchreport.RequestedReport{
			Mode:  matchreport.RequestedTerms,
			Terms: []yacymodel.Hash{related},
		},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if got := result.DocumentsMatchingEachReportedTerm[related]; !hasExactlyDocuments(
		got,
		"u2",
		"u3",
	) {
		t.Fatalf("DocumentsMatchingEachReportedTerm[related] = %v, want u2, u3", got)
	}
}

func TestSearchQualifiesByLanguageAndTermSpread(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	english := func(url string, position int) yacymodel.RWIPosting {
		posting := postingEntry(word1, url, position)
		posting.Language = mustLanguage(t, "en")

		return posting
	}
	inLanguage := func(word yacymodel.Hash, url, language string, position int) yacymodel.RWIPosting {
		posting := postingEntry(word, url, position)
		posting.Language = mustLanguage(t, language)

		return posting
	}

	near := english("u1", 1)
	nearOther := inLanguage(word2, "u1", "en", 2)
	german := inLanguage(word1, "u2", "de", 1)
	germanOther := inLanguage(word2, "u2", "de", 2)
	far := english("u3", 1)
	farOther := inLanguage(word2, "u3", "en", 9)

	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {near, german, far},
		word2: {nearOther, germanOther, farOther},
	}}
	results := searchresult.New(
		openVault(t),
		termpostings.New(index, 100),
		searchtest.URLDirectory{Documents: urlMetadata("u1", "u2", "u3")},
	)

	result, err := results.ResultFor(context.Background(), searchcriteria.Criteria{
		Terms:         []yacymodel.Hash{word1, word2},
		MaxTermSpread: 5,
		Language:      mustLanguage(t, "en"),
	}, matchreport.RequestedReport{})
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if len(result.DocumentMetadata) != 1 ||
		result.DocumentMetadata[0].Address != addressFor("u1") {
		t.Fatalf("resources = %v, want only u1", result.DocumentMetadata)
	}
}

func openVault(t *testing.T) *vault.Vault {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = v.Close() })

	return v
}
