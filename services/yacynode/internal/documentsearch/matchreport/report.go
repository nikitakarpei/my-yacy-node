// Package matchreport counts how many documents matched each term and lists the
// documents behind the terms a request asks to have reported.
package matchreport

import (
	"cmp"
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Mode int

const (
	NoMatches Mode = iota
	TermWithMostMatches
	RequestedTerms
)

type RequestedReport struct {
	Mode  Mode
	Terms []yacymodel.Hash
}

type Report struct {
	TotalMatchesPerTerm               map[yacymodel.Hash]int
	DocumentsMatchingEachReportedTerm map[yacymodel.Hash][]yacymodel.URLHash
}

func (r RequestedReport) ReportFor(
	ctx context.Context,
	index rwipostings.PostingIndex,
	maxPostingsPerTerm int,
	criteria searchcriteria.Criteria,
	matchesForQueryTerms map[yacymodel.Hash]termmatch.Match,
) (Report, error) {
	switch r.Mode {
	case NoMatches:
		return Report{}, nil
	case TermWithMostMatches:
		return reportForTermWithMostMatches(criteria, matchesForQueryTerms), nil
	case RequestedTerms:
		return r.reportForRequestedTerms(
			ctx,
			index,
			maxPostingsPerTerm,
			criteria,
			matchesForQueryTerms,
		)
	default:
		return Report{}, nil
	}
}

func reportForTermWithMostMatches(
	criteria searchcriteria.Criteria,
	matchesForQueryTerms map[yacymodel.Hash]termmatch.Match,
) Report {
	report := Report{TotalMatchesPerTerm: totalMatchesOf(matchesForQueryTerms)}
	if len(criteria.Terms) <= 1 || len(criteria.RequiredDocuments) != 0 {
		return report
	}
	term, ok := termWithMostMatches(matchesForQueryTerms)
	if !ok {
		return report
	}
	report.DocumentsMatchingEachReportedTerm = map[yacymodel.Hash][]yacymodel.URLHash{
		term: documentHashes(matchesForQueryTerms[term].PostingPerDocument),
	}

	return report
}

func totalMatchesOf(matches map[yacymodel.Hash]termmatch.Match) map[yacymodel.Hash]int {
	totals := make(map[yacymodel.Hash]int, len(matches))
	for term, match := range matches {
		totals[term] = match.TotalMatches
	}

	return totals
}

func termWithMostMatches(matches map[yacymodel.Hash]termmatch.Match) (yacymodel.Hash, bool) {
	var (
		mostMatchedTerm yacymodel.Hash
		mostMatches     int
		found           bool
	)
	for term, match := range matches {
		matchCount := len(match.PostingPerDocument)
		if !found || matchCount > mostMatches ||
			matchCount == mostMatches &&
				cmp.Compare(term.String(), mostMatchedTerm.String()) < 0 {
			mostMatchedTerm = term
			mostMatches = matchCount
			found = true
		}
	}

	return mostMatchedTerm, found
}

func documentHashes(
	postingPerDocument map[yacymodel.URLHash]termmatch.Posting,
) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(postingPerDocument))
	for documentHash := range postingPerDocument {
		hashes = append(hashes, documentHash)
	}

	return hashes
}

func (r RequestedReport) reportForRequestedTerms(
	ctx context.Context,
	index rwipostings.PostingIndex,
	maxPostingsPerTerm int,
	criteria searchcriteria.Criteria,
	matchesForQueryTerms map[yacymodel.Hash]termmatch.Match,
) (Report, error) {
	matchesForReportedTerms, err := termmatch.MatchesFor(
		ctx,
		index,
		r.Terms,
		postingfilter.FilterForReport(criteria),
		maxPostingsPerTerm,
	)
	if err != nil {
		return Report{}, err
	}

	documentsMatchingEachReportedTerm := make(
		map[yacymodel.Hash][]yacymodel.URLHash,
		len(r.Terms),
	)
	for _, term := range r.Terms {
		documentsMatchingEachReportedTerm[term] = documentHashes(
			matchesForReportedTerms[term].PostingPerDocument,
		)
	}

	totals := totalMatchesOf(matchesForQueryTerms)
	if len(criteria.Terms) == 0 {
		totals = totalMatchesOf(matchesForReportedTerms)
	}

	return Report{
		TotalMatchesPerTerm:               totals,
		DocumentsMatchingEachReportedTerm: documentsMatchingEachReportedTerm,
	}, nil
}
