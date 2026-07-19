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
	index          rwipostings.PostingIndex
	documents      urlmeta.URLDirectory
	matchesPerTerm int
}

type searchResult struct {
	resources                         []yacymodel.URLMetadata
	topics                            []string
	totalDocumentsMatchingEveryTerm   int
	searchDuration                    time.Duration
	totalMatchesPerTerm               map[yacymodel.Hash]int
	documentsMatchingEachReportedTerm map[yacymodel.Hash]string
}

func (s searcher) search(ctx context.Context, criteria searchCriteria) (searchResult, error) {
	start := time.Now()

	filter, err := s.postingFilter(ctx, criteria, criteria.excludedTerms)
	if err != nil {
		return searchResult{}, err
	}
	wanted, err := s.documentsMatchingTerms(ctx, criteria.terms, filter)
	if err != nil {
		return searchResult{}, err
	}

	matchingEveryTerm := documentsWithinTermSpread(
		keepDocumentsMatchingEveryTerm(
			documentsInTermOrder(criteria.terms, wanted.documentsPerTerm),
		),
		criteria.maxTermSpread,
		len(criteria.terms),
	)
	mostRelevant := takeMostRelevant(
		documentsOrderedByRelevance(matchingEveryTerm, len(criteria.terms)),
		criteria.maxResults,
	)
	resources, err := s.documents.MetadataByHash(ctx, mostRelevant)
	if err != nil {
		return searchResult{}, fmt.Errorf("rows by hash: %w", err)
	}

	report, err := s.reportMatches(ctx, criteria, wanted)
	if err != nil {
		return searchResult{}, err
	}

	return searchResult{
		resources:                         resources,
		topics:                            resultTopics(resources, criteria.terms),
		totalDocumentsMatchingEveryTerm:   len(matchingEveryTerm),
		searchDuration:                    time.Since(start),
		totalMatchesPerTerm:               report.totalMatchesPerTerm,
		documentsMatchingEachReportedTerm: report.documents,
	}, nil
}
