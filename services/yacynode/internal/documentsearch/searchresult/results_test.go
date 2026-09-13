package searchresult_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type searchIndex interface {
	rwipostings.PostingIndex
	rwipostingimpactorder.ImpactOrderQuery
	rwipostingamount.PostingAmountQuery
}

const mostRelevantDocumentsPerTerm = 1000

func resultsFor(
	t *testing.T,
	index searchIndex,
	documents searchresult.DocumentDirectory,
) searchresult.Results {
	t.Helper()

	return searchresult.New(
		openVault(t),
		documentmatch.New(index, index),
		termmatch.New(index, index, mostRelevantDocumentsPerTerm),
		index,
		documents,
	)
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

func mustLanguage(t *testing.T, raw string) yacymodel.Language {
	t.Helper()

	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		t.Fatalf("parse language %q: %v", raw, err)
	}

	return language
}

func postingEntry(word yacymodel.Hash, document string, position int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:     word,
		URLHash:      searchtest.URLHashFor(document),
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
	results := resultsFor(t, index, searchtest.URLDirectory{
		Documents: urlMetadata("u1", "u2", "u3"),
	})

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}, MaxResults: 10},
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractOfTermWithMostPostings{},
		},
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
	if len(result.MatchedDocuments) != 1 {
		t.Fatalf("resources = %d, want 1", len(result.MatchedDocuments))
	}
	if result.MatchedDocuments[0].Metadata.Address != addressFor("u2") {
		t.Errorf("resource = %v, want u2", result.MatchedDocuments[0].Metadata)
	}
	if got := result.IndexAbstracts[word1]; !hasExactlyDocuments(got, "u1", "u2") {
		t.Errorf("IndexAbstracts[w1] = %v, want u1, u2", got)
	}
}

func TestSearchReportsTheAmountOfPostingsItHoldsWithoutReadingThemAll(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {
			postingEntry(word, "u1", 0),
			postingEntry(word, "u2", 0),
			postingEntry(word, "u3", 0),
		},
	}}
	results := resultsFor(t, index, searchtest.URLDirectory{
		Documents: urlMetadata("u1", "u2", "u3"),
	})

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}, MaxResults: 1},
		nil,
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if result.AmountOfPostingsPerTerm[word] != 3 {
		t.Errorf(
			"AmountOfPostingsPerTerm[w1] = %d, want the 3 postings the node holds",
			result.AmountOfPostingsPerTerm[word],
		)
	}
	if len(result.MatchedDocuments) != 1 {
		t.Errorf("resources = %d, want 1", len(result.MatchedDocuments))
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
	results := resultsFor(t, index, searchtest.URLDirectory{
		Documents: urlMetadata("u1", "u2", "u3"),
	})

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}, MaxResults: 2},
		nil,
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if len(result.MatchedDocuments) != 2 {
		t.Errorf("resources = %d, want 2", len(result.MatchedDocuments))
	}
}

func TestSearchFiltersByAverageGapNotSpan(t *testing.T) {
	word1, word2, word3 := searchtest.HashFor("w1"),
		searchtest.HashFor("w2"),
		searchtest.HashFor("w3")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "uA", 1), postingEntry(word1, "uB", 1)},
		word2: {postingEntry(word2, "uA", 5), postingEntry(word2, "uB", 10)},
		word3: {postingEntry(word3, "uA", 9), postingEntry(word3, "uB", 20)},
	}}
	results := resultsFor(t, index, searchtest.URLDirectory{Documents: urlMetadata("uA", "uB")})

	result, err := results.ResultFor(context.Background(), searchcriteria.Criteria{
		Terms:         []yacymodel.Hash{word1, word2, word3},
		MaxTermSpread: 5,
		MaxResults:    10,
	}, nil)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if len(result.MatchedDocuments) != 1 ||
		result.MatchedDocuments[0].Metadata.Address != addressFor("uA") {
		t.Fatalf("resources = %v, want only uA (span 8, average gap 4)", result.MatchedDocuments)
	}
}

func TestSearchSurfacesQueryTermFailures(t *testing.T) {
	results := resultsFor(
		t,
		searchtest.FailingPostingIndex{Err: errIndexBroken},
		searchtest.URLDirectory{},
	)

	_, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{searchtest.HashFor("w1")}},
		nil,
	)
	if !errors.Is(err, errIndexBroken) {
		t.Fatalf("ResultFor error = %v, want %v", err, errIndexBroken)
	}
}

func TestSearchSurfacesMetadataFailures(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1", 0)},
	}}
	results := resultsFor(t, index, searchtest.FailingURLDirectory{Err: errDirectoryBroken})

	_, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}, MaxResults: 10},
		nil,
	)
	if !errors.Is(err, errDirectoryBroken) {
		t.Fatalf("ResultFor error = %v, want %v", err, errDirectoryBroken)
	}
}

func TestSearchSurfacesIndexAbstractTermFailures(t *testing.T) {
	results := resultsFor(
		t,
		searchtest.FailingPostingIndex{Err: errIndexBroken},
		searchtest.URLDirectory{},
	)

	_, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{},
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{searchtest.HashFor("w2")}},
		},
	)
	if !errors.Is(err, errIndexBroken) {
		t.Fatalf("ResultFor error = %v, want %v", err, errIndexBroken)
	}
}

var (
	errIndexBroken     = errors.New("index broken")
	errDirectoryBroken = errors.New("directory broken")
)

func TestSearchAbstractsRequestedTermsWithoutQueryTerms(t *testing.T) {
	word := searchtest.HashFor("w1")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1", 1), postingEntry(word, "u2", 1)},
	}}
	results := resultsFor(t, index, searchtest.URLDirectory{})

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{},
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{word}},
		},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if result.TotalDocumentsMatchingEveryTerm != 0 || len(result.MatchedDocuments) != 0 {
		t.Fatalf("result = %+v, want report only", result)
	}
	if len(result.AmountOfPostingsPerTerm) != 0 {
		t.Errorf("AmountOfPostingsPerTerm = %v, want none without query terms",
			result.AmountOfPostingsPerTerm)
	}
	if got := result.IndexAbstracts[word]; !hasExactlyDocuments(got, "u1", "u2") {
		t.Errorf("IndexAbstracts = %v, want u1, u2", got)
	}
}

func TestSearchAbstractsRequestedTermsAlongsideQueryTerms(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word:    {postingEntry(word, "u1", 0), postingEntry(word, "u2", 0)},
		related: {postingEntry(related, "u2", 0), postingEntry(related, "u3", 0)},
	}}
	results := resultsFor(t, index, searchtest.URLDirectory{Documents: urlMetadata("u1", "u2")})

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}, MaxResults: 10},
		indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: []yacymodel.Hash{related}},
		},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if got := result.IndexAbstracts[related]; !hasExactlyDocuments(got, "u2", "u3") {
		t.Fatalf("IndexAbstracts[related] = %v, want u2, u3", got)
	}
}

func TestSearchQualifiesByLanguageAndTermSpread(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	inLanguage := func(
		word yacymodel.Hash,
		document, language string,
		position int,
	) yacymodel.RWIPosting {
		posting := postingEntry(word, document, position)
		posting.Language = mustLanguage(t, language)

		return posting
	}

	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {
			inLanguage(word1, "u1", "en", 1),
			inLanguage(word1, "u2", "de", 1),
			inLanguage(word1, "u3", "en", 1),
		},
		word2: {
			inLanguage(word2, "u1", "en", 2),
			inLanguage(word2, "u2", "de", 2),
			inLanguage(word2, "u3", "en", 9),
		},
	}}
	results := resultsFor(t, index, searchtest.URLDirectory{
		Documents: urlMetadata("u1", "u2", "u3"),
	})

	result, err := results.ResultFor(context.Background(), searchcriteria.Criteria{
		Terms:         []yacymodel.Hash{word1, word2},
		MaxTermSpread: 5,
		MaxResults:    10,
		Language:      yacymodel.Some(mustLanguage(t, "en")),
	}, nil)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if len(result.MatchedDocuments) != 1 ||
		result.MatchedDocuments[0].Metadata.Address != addressFor("u1") {
		t.Fatalf("resources = %v, want only u1", result.MatchedDocuments)
	}
}
