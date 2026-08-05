package documentsearch

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type searcher struct {
	index              rwipostings.PostingIndex
	documentDirectory  urlmeta.URLDirectory
	maxPostingsPerTerm int
}

type searchResult struct {
	documentMetadata                  []yacymodel.URLMetadata
	topics                            []string
	totalDocumentsMatchingEveryTerm   int
	searchDuration                    time.Duration
	totalMatchesPerTerm               map[yacymodel.Hash]int
	documentsMatchingEachReportedTerm map[yacymodel.Hash][]yacymodel.URLHash
}

func (s searcher) search(
	ctx context.Context,
	criteria searchCriteria,
	requestedReport requestedMatchReport,
) (searchResult, error) {
	start := time.Now()

	filter, err := s.searchPostingFilterFrom(ctx, criteria)
	if err != nil {
		return searchResult{}, err
	}
	matchesForQueryTerms, err := s.termMatchesFor(ctx, criteria.terms, filter)
	if err != nil {
		return searchResult{}, err
	}

	matchesAcrossEveryTerm := documentMatchesAcrossEveryTerm(criteria.terms, matchesForQueryTerms)
	matchesWithinTermSpread := documentMatchesWithinTermSpread(
		matchesAcrossEveryTerm,
		criteria.maxTermSpread,
		len(criteria.terms),
	)
	documentHashes := hashesOfMostRelevantDocuments(
		matchesWithinTermSpread,
		len(criteria.terms),
		criteria.maxResults,
	)
	documentMetadata, err := s.documentDirectory.MetadataByHash(ctx, documentHashes)
	if err != nil {
		return searchResult{}, fmt.Errorf("document metadata: %w", err)
	}

	report, err := s.matchReportFor(ctx, criteria, requestedReport, matchesForQueryTerms)
	if err != nil {
		return searchResult{}, err
	}

	return searchResult{
		documentMetadata:                  documentMetadata,
		topics:                            topicsFromTitles(documentMetadata, criteria.terms),
		totalDocumentsMatchingEveryTerm:   len(matchesWithinTermSpread),
		searchDuration:                    time.Since(start),
		totalMatchesPerTerm:               report.totalMatchesPerTerm,
		documentsMatchingEachReportedTerm: report.documentsMatchingEachReportedTerm,
	}, nil
}
