package documentsearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type matchReportMode int

const (
	reportNoMatches matchReportMode = iota
	reportTermWithMostMatches
	reportRequestedTerms
)

type requestedMatchReport struct {
	mode  matchReportMode
	terms []yacymodel.Hash
}

func requestedMatchReportFrom(req yacyproto.SearchRequest) requestedMatchReport {
	switch req.Abstracts {
	case "":
		return requestedMatchReport{mode: reportNoMatches}
	case yacyproto.SearchAbstractsAuto:
		return requestedMatchReport{mode: reportTermWithMostMatches}
	default:
		return requestedMatchReport{mode: reportRequestedTerms, terms: req.Abstracts.Hashes()}
	}
}

type matchReport struct {
	totalMatchesPerTerm               map[yacymodel.Hash]int
	documentsMatchingEachReportedTerm map[yacymodel.Hash][]yacymodel.URLHash
}

func (s searcher) matchReportFor(
	ctx context.Context,
	criteria searchCriteria,
	requestedReport requestedMatchReport,
	matchesForQueryTerms map[yacymodel.Hash]termMatch,
) (matchReport, error) {
	switch requestedReport.mode {
	case reportNoMatches:
		return matchReport{}, nil
	case reportTermWithMostMatches:
		return matchReportForTermWithMostMatches(criteria, matchesForQueryTerms), nil
	case reportRequestedTerms:
		return s.matchReportForRequestedTerms(
			ctx,
			criteria,
			requestedReport.terms,
			matchesForQueryTerms,
		)
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
	reportedTerms []yacymodel.Hash,
	matchesForQueryTerms map[yacymodel.Hash]termMatch,
) (matchReport, error) {
	matchesForReportedTerms, err := s.termMatchesFor(
		ctx,
		reportedTerms,
		reportPostingFilterFrom(criteria),
	)
	if err != nil {
		return matchReport{}, err
	}

	documentsMatchingEachReportedTerm := make(
		map[yacymodel.Hash][]yacymodel.URLHash,
		len(reportedTerms),
	)
	for _, term := range reportedTerms {
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
