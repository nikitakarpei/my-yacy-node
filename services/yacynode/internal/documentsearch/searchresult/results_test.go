package searchresult

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/matchreport"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
)

type fakeScanner struct {
	postings map[yacymodel.Hash][]yacymodel.RWIPosting
}

func (s fakeScanner) RWICount(context.Context) (int, error) {
	return len(s.postings), nil
}

func (s fakeScanner) ScanWord(
	_ context.Context,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	for _, entry := range s.postings[word] {
		entry.WordHash = word
		keepGoing, err := visit(entry)
		if err != nil {
			return err
		}
		if !keepGoing {
			return nil
		}
	}

	return nil
}

func (s fakeScanner) PostingOf(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	for _, entry := range s.postings[word] {
		if entry.URLHash == url {
			entry.WordHash = word

			return entry, true, nil
		}
	}

	return yacymodel.RWIPosting{}, false, nil
}

type fakeDirectory struct {
	documentDirectory map[yacymodel.URLHash]yacymodel.URLMetadata
}

func (d fakeDirectory) MetadataByHash(
	_ context.Context,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	out := make([]yacymodel.URLMetadata, 0, len(hashes))
	for _, hash := range hashes {
		if stored, ok := d.documentDirectory[hash]; ok {
			out = append(out, stored)
		}
	}

	return out, nil
}

func (d fakeDirectory) MissingURLs(
	context.Context,
	[]yacymodel.URLHash,
) ([]yacymodel.URLHash, error) {
	return nil, nil
}

func (d fakeDirectory) Count(context.Context) (int, error) {
	return len(d.documentDirectory), nil
}

func hashFor(base string) yacymodel.Hash {
	const filler = "AAAAAAAAAAAA"
	padded := base + filler[len(base):]
	if len(base) >= yacymodel.HashLength {
		padded = base[:yacymodel.HashLength]
	}
	hash, err := yacymodel.ParseHash(padded)
	if err != nil {
		panic(err)
	}

	return hash
}

func mustLanguage(t *testing.T, raw string) yacymodel.Optional[yacymodel.Language] {
	t.Helper()

	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		t.Fatalf("parse language %q: %v", raw, err)
	}

	return yacymodel.Some(language)
}

func urlHashFor(url string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(hashFor(url).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func postingEntry(
	word yacymodel.Hash,
	url string,
	position int,
	hits int,
) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:     word,
		URLHash:      urlHashFor(url),
		Hits:         hits,
		TextPosition: position,
	}
}

func addressFor(id string) string {
	return "http://example.com/" + id
}

func urlMetadata(ids ...string) map[yacymodel.URLHash]yacymodel.URLMetadata {
	metadata := make(map[yacymodel.URLHash]yacymodel.URLMetadata, len(ids))
	for _, id := range ids {
		metadata[urlHashFor(id)] = yacymodel.URLMetadata{Address: addressFor(id)}
	}

	return metadata
}

func hasExactlyDocuments(got []yacymodel.URLHash, ids ...string) bool {
	if len(got) != len(ids) {
		return false
	}
	want := make(map[yacymodel.URLHash]struct{}, len(ids))
	for _, id := range ids {
		want[urlHashFor(id)] = struct{}{}
	}
	for _, hash := range got {
		if _, ok := want[hash]; !ok {
			return false
		}
	}

	return true
}

func TestSearchJoinsAndCountsAndReports(t *testing.T) {
	word1, word2 := hashFor("w1"), hashFor("w2")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "u1", 0, 1), postingEntry(word1, "u2", 0, 1)},
		word2: {postingEntry(word2, "u2", 0, 1), postingEntry(word2, "u3", 0, 1)},
	}}
	results := New(
		index,
		fakeDirectory{documentDirectory: urlMetadata("u1", "u2", "u3")},
		100,
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
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {
			postingEntry(word, "u1", 0, 1),
			postingEntry(word, "u2", 0, 1),
			postingEntry(word, "u3", 0, 1),
		},
	}}
	results := New(
		index,
		fakeDirectory{documentDirectory: urlMetadata("u1", "u2", "u3")},
		100,
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

func TestSearchOrdersByOccurrencesThenTermSpread(t *testing.T) {
	word1, word2 := hashFor("w1"), hashFor("w2")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "u2", 1, 1), postingEntry(word1, "u3", 1, 1)},
		word2: {postingEntry(word2, "u2", 2, 2), postingEntry(word2, "u3", 5, 2)},
	}}
	results := New(index, fakeDirectory{documentDirectory: urlMetadata("u2", "u3")}, 100)

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
		matchreport.RequestedReport{},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if got := result.DocumentMetadata[0].Address; got != addressFor("u2") {
		t.Errorf("first resource = %q, want u2", got)
	}
}

func TestSearchFiltersByAverageGapNotSpan(t *testing.T) {
	word1, word2, word3 := hashFor("w1"), hashFor("w2"), hashFor("w3")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {postingEntry(word1, "uA", 1, 1), postingEntry(word1, "uB", 1, 1)},
		word2: {postingEntry(word2, "uA", 5, 1), postingEntry(word2, "uB", 10, 1)},
		word3: {postingEntry(word3, "uA", 9, 1), postingEntry(word3, "uB", 20, 1)},
	}}
	results := New(index, fakeDirectory{documentDirectory: urlMetadata("uA", "uB")}, 100)

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

func TestSearchCapKeepsMostFrequentPostings(t *testing.T) {
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1", 1, 1), postingEntry(word, "u2", 1, 5)},
	}}
	results := New(index, fakeDirectory{documentDirectory: urlMetadata("u1", "u2")}, 1)

	result, err := results.ResultFor(
		context.Background(),
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		matchreport.RequestedReport{},
	)
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if len(result.DocumentMetadata) != 1 ||
		result.DocumentMetadata[0].Address != addressFor("u2") {
		t.Fatalf(
			"resources = %v, want only u2 (highest hit count kept under cap)",
			result.DocumentMetadata,
		)
	}
}

func TestSearchExcludesTerms(t *testing.T) {
	word, ban := hashFor("w1"), hashFor("ban")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1", 0, 1), postingEntry(word, "u2", 0, 1)},
		ban:  {postingEntry(ban, "u2", 0, 1)},
	}}
	results := New(index, fakeDirectory{documentDirectory: urlMetadata("u1", "u2")}, 100)

	result, err := results.ResultFor(context.Background(), searchcriteria.Criteria{
		Terms:         []yacymodel.Hash{word},
		ExcludedTerms: []yacymodel.Hash{ban},
	}, matchreport.RequestedReport{})
	if err != nil {
		t.Fatalf("ResultFor: %v", err)
	}
	if len(result.DocumentMetadata) != 1 ||
		result.DocumentMetadata[0].Address != addressFor("u1") {
		t.Fatalf("resources = %v, want only u1", result.DocumentMetadata)
	}
}

func TestSearchReportsRequestedTermsWithoutWantedTerms(t *testing.T) {
	word := hashFor("w1")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word: {postingEntry(word, "u1", 1, 1), postingEntry(word, "u2", 1, 1)},
	}}
	results := New(index, fakeDirectory{}, 100)

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
	word, related := hashFor("w1"), hashFor("w2")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word:    {postingEntry(word, "u1", 0, 1), postingEntry(word, "u2", 0, 1)},
		related: {postingEntry(related, "u2", 0, 1), postingEntry(related, "u3", 0, 1)},
	}}
	results := New(index, fakeDirectory{documentDirectory: urlMetadata("u1", "u2")}, 100)

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
	word1, word2 := hashFor("w1"), hashFor("w2")
	english := func(url string, position int) yacymodel.RWIPosting {
		posting := postingEntry(word1, url, position, 1)
		posting.Language = mustLanguage(t, "en")

		return posting
	}
	inLanguage := func(word yacymodel.Hash, url, language string, position int) yacymodel.RWIPosting {
		posting := postingEntry(word, url, position, 1)
		posting.Language = mustLanguage(t, language)

		return posting
	}

	near := english("u1", 1)
	nearOther := inLanguage(word2, "u1", "en", 2)
	german := inLanguage(word1, "u2", "de", 1)
	germanOther := inLanguage(word2, "u2", "de", 2)
	far := english("u3", 1)
	farOther := inLanguage(word2, "u3", "en", 9)

	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		word1: {near, german, far},
		word2: {nearOther, germanOther, farOther},
	}}
	results := New(
		index,
		fakeDirectory{documentDirectory: urlMetadata("u1", "u2", "u3")},
		100,
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
