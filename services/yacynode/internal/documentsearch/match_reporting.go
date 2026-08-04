package documentsearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type matchReport struct {
	totalMatchesPerTerm               map[yacymodel.Hash]int
	documentsMatchingEachReportedTerm map[yacymodel.Hash][]yacymodel.URLHash
}

func (s searcher) matchReportFor(
	ctx context.Context,
	criteria searchCriteria,
	matchesForQueryTerms map[yacymodel.Hash]termMatch,
) (matchReport, error) {
	switch criteria.requestedReport.mode {
	case reportNoMatches:
		return matchReport{}, nil
	case reportTermWithMostMatches:
		return matchReportForTermWithMostMatches(criteria, matchesForQueryTerms), nil
	case reportRequestedTerms:
		return s.matchReportForRequestedTerms(ctx, criteria, matchesForQueryTerms)
	default:
		return matchReport{}, nil
	}
}

func matchReportForTermWithMostMatches(
	criteria searchCriteria,
	matchesForQueryTerms map[yacymodel.Hash]termMatch,
) matchReport {
	report := matchReport{totalMatchesPerTerm: totalPostingsOf(matchesForQueryTerms)}
	if len(criteria.terms) <= 1 || len(criteria.requiredDocuments) != 0 {
		return report
	}
	term, ok := termWithMostMatches(matchesForQueryTerms)
	if !ok {
		return report
	}
	report.documentsMatchingEachReportedTerm = map[yacymodel.Hash][]yacymodel.URLHash{
		term: documentHashes(matchesForQueryTerms[term].postingPerDocument),
	}

	return report
}

func (s searcher) matchReportForRequestedTerms(
	ctx context.Context,
	criteria searchCriteria,
	matchesForQueryTerms map[yacymodel.Hash]termMatch,
) (matchReport, error) {
	filter, err := s.postingFilter(ctx, criteria, nil)
	if err != nil {
		return matchReport{}, err
	}
	matchesForReportedTerms, err := s.termMatchesFor(ctx, criteria.requestedReport.terms, filter)
	if err != nil {
		return matchReport{}, err
	}

	documentsMatchingEachReportedTerm := make(
		map[yacymodel.Hash][]yacymodel.URLHash,
		len(criteria.requestedReport.terms),
	)
	for _, term := range criteria.requestedReport.terms {
		documentsMatchingEachReportedTerm[term] = documentHashes(
			matchesForReportedTerms[term].postingPerDocument,
		)
	}

	totals := totalPostingsOf(matchesForQueryTerms)
	if len(criteria.terms) == 0 {
		totals = totalPostingsOf(matchesForReportedTerms)
	}

	return matchReport{
		totalMatchesPerTerm:               totals,
		documentsMatchingEachReportedTerm: documentsMatchingEachReportedTerm,
	}, nil
}

func termWithMostMatches(matches map[yacymodel.Hash]termMatch) (yacymodel.Hash, bool) {
	var (
		mostMatchedTerm yacymodel.Hash
		mostMatches     int
		found           bool
	)
	for term, match := range matches {
		matchCount := len(match.postingPerDocument)
		if !found || matchCount > mostMatches ||
			matchCount == mostMatches &&
				compareAscending(term.String(), mostMatchedTerm.String()) < 0 {
			mostMatchedTerm = term
			mostMatches = matchCount
			found = true
		}
	}

	return mostMatchedTerm, found
}
