package matchreport_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/matchreport"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

func matchOf(urls ...string) termpostings.Match {
	byDocument := make(map[yacymodel.URLHash]termpostings.Posting, len(urls))
	for _, url := range urls {
		byDocument[searchtest.URLHashFor(url)] = termpostings.Posting{
			DocumentHash: searchtest.URLHashFor(url),
		}
	}

	return termpostings.Match{PostingPerDocument: byDocument, TotalMatches: len(urls)}
}

func TestReportForNoMatchesStaysEmpty(t *testing.T) {
	report, err := matchreport.RequestedReport{Mode: matchreport.NoMatches}.ReportFor(
		context.Background(),
		nil,
		stubTermMatches{},
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
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	matches := map[yacymodel.Hash]termpostings.Match{
		word1: matchOf("u1", "u2"),
		word2: matchOf("u2"),
	}

	report, err := matchreport.RequestedReport{Mode: matchreport.TermWithMostMatches}.ReportFor(
		context.Background(),
		nil,
		stubTermMatches{},
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
	word := searchtest.HashFor("w1")

	report, err := matchreport.RequestedReport{Mode: matchreport.TermWithMostMatches}.ReportFor(
		context.Background(),
		nil,
		stubTermMatches{},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		map[yacymodel.Hash]termpostings.Match{word: matchOf("u1")},
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if report.DocumentsMatchingEachReportedTerm != nil {
		t.Fatalf("reported documents = %v, want none", report.DocumentsMatchingEachReportedTerm)
	}
}

func TestReportForTermWithMostMatchesCountsOnlyWithRequiredDocuments(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	report, err := matchreport.RequestedReport{Mode: matchreport.TermWithMostMatches}.ReportFor(
		context.Background(),
		nil,
		stubTermMatches{},
		searchcriteria.Criteria{
			Terms:             []yacymodel.Hash{word1, word2},
			RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("u1")},
		},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchOf("u1", "u2"),
			word2: matchOf("u2"),
		},
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if report.TotalMatchesPerTerm[word1] != 2 {
		t.Errorf("TotalMatchesPerTerm[w1] = %d, want 2", report.TotalMatchesPerTerm[word1])
	}
	if report.DocumentsMatchingEachReportedTerm != nil {
		t.Fatalf("reported documents = %v, want none", report.DocumentsMatchingEachReportedTerm)
	}
}

func TestReportForTermWithMostMatchesBreaksTiesBySmallerTerm(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	report, err := matchreport.RequestedReport{Mode: matchreport.TermWithMostMatches}.ReportFor(
		context.Background(),
		nil,
		stubTermMatches{},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
		map[yacymodel.Hash]termpostings.Match{
			word1: matchOf("u1"),
			word2: matchOf("u2"),
		},
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if _, ok := report.DocumentsMatchingEachReportedTerm[word1]; !ok {
		t.Fatalf(
			"reported documents = %v, want the tie broken toward w1",
			report.DocumentsMatchingEachReportedTerm,
		)
	}
	if _, ok := report.DocumentsMatchingEachReportedTerm[word2]; ok {
		t.Fatalf(
			"reported documents = %v, want only w1",
			report.DocumentsMatchingEachReportedTerm,
		)
	}
}

func TestReportForTermWithMostMatchesCountsOnlyWithoutMatches(t *testing.T) {
	word1, word2 := searchtest.HashFor("w1"), searchtest.HashFor("w2")

	report, err := matchreport.RequestedReport{Mode: matchreport.TermWithMostMatches}.ReportFor(
		context.Background(),
		nil,
		stubTermMatches{},
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word1, word2}},
		nil,
	)
	if err != nil {
		t.Fatalf("ReportFor: %v", err)
	}
	if len(report.TotalMatchesPerTerm) != 0 {
		t.Errorf("TotalMatchesPerTerm = %v, want empty", report.TotalMatchesPerTerm)
	}
	if report.DocumentsMatchingEachReportedTerm != nil {
		t.Fatalf("reported documents = %v, want none", report.DocumentsMatchingEachReportedTerm)
	}
}

func TestReportForRequestedTermsScansThoseTerms(t *testing.T) {
	word, related := searchtest.HashFor("w1"), searchtest.HashFor("w2")
	reportedMatches := stubTermMatches{matchPerTerm: map[yacymodel.Hash]termpostings.Match{
		related: matchOf("u2", "u3"),
	}}

	report, err := matchreport.RequestedReport{
		Mode:  matchreport.RequestedTerms,
		Terms: []yacymodel.Hash{related},
	}.ReportFor(
		context.Background(),
		nil,
		reportedMatches,
		searchcriteria.Criteria{Terms: []yacymodel.Hash{word}},
		map[yacymodel.Hash]termpostings.Match{word: matchOf("u1")},
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

func TestReportForRequestedTermsSurfacesScanFailures(t *testing.T) {
	_, err := matchreport.RequestedReport{
		Mode:  matchreport.RequestedTerms,
		Terms: []yacymodel.Hash{searchtest.HashFor("w2")},
	}.ReportFor(
		context.Background(),
		nil,
		stubTermMatches{err: errScanBroken},
		searchcriteria.Criteria{},
		nil,
	)
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("ReportFor error = %v, want %v", err, errScanBroken)
	}
}

var errScanBroken = errors.New("scan broken")

func TestReportForRequestedTermsCountsThemWithoutQueryTerms(t *testing.T) {
	related := searchtest.HashFor("w2")
	reportedMatches := stubTermMatches{matchPerTerm: map[yacymodel.Hash]termpostings.Match{
		related: matchOf("u2"),
	}}

	report, err := matchreport.RequestedReport{
		Mode:  matchreport.RequestedTerms,
		Terms: []yacymodel.Hash{related},
	}.ReportFor(
		context.Background(),
		nil,
		reportedMatches,
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

type stubTermMatches struct {
	matchPerTerm map[yacymodel.Hash]termpostings.Match
	err          error
}

func (s stubTermMatches) MatchesFor(
	_ context.Context,
	_ *vault.Txn,
	terms []yacymodel.Hash,
	_ postingfilter.Filter,
) (map[yacymodel.Hash]termpostings.Match, error) {
	if s.err != nil {
		return nil, s.err
	}

	matches := make(map[yacymodel.Hash]termpostings.Match, len(terms))
	for _, term := range terms {
		matches[term] = s.matchPerTerm[term]
	}

	return matches, nil
}
