package matchreport

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termmatch"
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

func hashFor(base string) yacymodel.Hash {
	const filler = "AAAAAAAAAAAA"
	hash, err := yacymodel.ParseHash(base + filler[len(base):])
	if err != nil {
		panic(err)
	}

	return hash
}

func urlHashFor(url string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(hashFor(url).String())
	if err != nil {
		panic(err)
	}

	return hash
}

func matchOf(urls ...string) termmatch.Match {
	byDocument := make(map[yacymodel.URLHash]termmatch.Posting, len(urls))
	for _, url := range urls {
		byDocument[urlHashFor(url)] = termmatch.Posting{DocumentHash: urlHashFor(url)}
	}

	return termmatch.Match{PostingPerDocument: byDocument, TotalMatches: len(urls)}
}

func TestReportForNoMatchesStaysEmpty(t *testing.T) {
	report, err := RequestedReport{Mode: NoMatches}.ReportFor(
		context.Background(),
		fakeScanner{},
		100,
		searchcriteria.Criteria{},
		nil,
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if report.TotalMatchesPerTerm != nil || report.DocumentsMatchingEachReportedTerm != nil {
		t.Fatalf("report = %+v, want empty", report)
	}
}

func TestReportForTermWithMostMatchesNamesTheWidestTerm(t *testing.T) {
	word1, word2 := hashFor("w1"), hashFor("w2")
	matches := map[yacymodel.Hash]termmatch.Match{
		word1: matchOf("u1", "u2"),
		word2: matchOf("u2"),
	}

	report, err := RequestedReport{Mode: TermWithMostMatches}.ReportFor(
		context.Background(),
		fakeScanner{},
		100,
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
		matches,
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if report.TotalMatchesPerTerm[word1] != 2 {
		t.Errorf("TotalMatchesPerTerm[w1] = %d, want 2", report.TotalMatchesPerTerm[word1])
	}
	if len(report.DocumentsMatchingEachReportedTerm[word1]) != 2 {
		t.Fatalf(
			"reported documents = %v, want the two behind w1",
			report.DocumentsMatchingEachReportedTerm,
		)
	}
}

func TestReportForTermWithMostMatchesCountsOnlyOnSingleTerm(t *testing.T) {
	word := hashFor("w1")

	report, err := RequestedReport{Mode: TermWithMostMatches}.ReportFor(
		context.Background(),
		fakeScanner{},
		100,
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		map[yacymodel.Hash]termmatch.Match{word: matchOf("u1")},
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if report.DocumentsMatchingEachReportedTerm != nil {
		t.Fatalf("reported documents = %v, want none", report.DocumentsMatchingEachReportedTerm)
	}
}

func TestReportForRequestedTermsScansThoseTerms(t *testing.T) {
	word, related := hashFor("w1"), hashFor("w2")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		related: {
			{URLHash: urlHashFor("u2"), Hits: 1},
			{URLHash: urlHashFor("u3"), Hits: 1},
		},
	}}

	report, err := RequestedReport{
		Mode:  RequestedTerms,
		Terms: []yacymodel.Hash{related},
	}.ReportFor(
		context.Background(),
		index,
		100,
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		map[yacymodel.Hash]termmatch.Match{word: matchOf("u1")},
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if report.TotalMatchesPerTerm[word] != 1 {
		t.Errorf("TotalMatchesPerTerm[w1] = %d, want 1", report.TotalMatchesPerTerm[word])
	}
	if len(report.DocumentsMatchingEachReportedTerm[related]) != 2 {
		t.Fatalf(
			"reported documents = %v, want the two behind w2",
			report.DocumentsMatchingEachReportedTerm,
		)
	}
}

func TestReportForRequestedTermsCountsThemWithoutQueryTerms(t *testing.T) {
	related := hashFor("w2")
	index := fakeScanner{postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		related: {{URLHash: urlHashFor("u2"), Hits: 1}},
	}}

	report, err := RequestedReport{
		Mode:  RequestedTerms,
		Terms: []yacymodel.Hash{related},
	}.ReportFor(
		context.Background(),
		index,
		100,
		searchcriteria.Criteria{},
		nil,
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if report.TotalMatchesPerTerm[related] != 1 {
		t.Errorf("TotalMatchesPerTerm[w2] = %d, want 1", report.TotalMatchesPerTerm[related])
	}
}
