// Package searchresult runs one search pass: it joins the query terms over the
// posting index, reads the metadata of the most relevant documents, and adds the
// match report the request asked for.
package searchresult

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/matchreport"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/titletopics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type Results struct {
	index              rwipostings.PostingIndex
	documentDirectory  urlmeta.URLDirectory
	maxPostingsPerTerm int
}

func New(
	index rwipostings.PostingIndex,
	documents urlmeta.URLDirectory,
	maxPostingsPerTerm int,
) Results {
	return Results{
		index:              index,
		documentDirectory:  documents,
		maxPostingsPerTerm: maxPostingsPerTerm,
	}
}

type Result struct {
	DocumentMetadata                  []yacymodel.URLMetadata
	Topics                            []string
	TotalDocumentsMatchingEveryTerm   int
	Duration                          time.Duration
	TotalMatchesPerTerm               map[yacymodel.Hash]int
	PostingsHeldPerTerm               map[yacymodel.Hash]int
	DocumentsMatchingEachReportedTerm map[yacymodel.Hash][]yacymodel.URLHash
}

var ErrDocumentDirectory = errors.New("document metadata")

func (r Results) ResultFor(
	ctx context.Context,
	criteria searchcriteria.Criteria,
	requestedReport matchreport.RequestedReport,
) (Result, error) {
	start := time.Now()

	filter, err := postingfilter.FilterForSearch(ctx, r.index, criteria)
	if err != nil {
		return Result{}, err
	}
	matchesForQueryTerms, err := termmatch.MatchesFor(
		ctx,
		criteria.Terms,
		r.index,
		filter,
		r.maxPostingsPerTerm,
	)
	if err != nil {
		return Result{}, err
	}

	matchesAcrossEveryTerm := documentmatch.MatchesAcrossEveryTerm(
		criteria.Terms,
		matchesForQueryTerms,
	)
	matchesWithinTermSpread := documentmatch.MatchesWithinTermSpread(
		matchesAcrossEveryTerm,
		criteria.MaxTermSpread,
		len(criteria.Terms),
	)
	documentHashes := documentmatch.HashesOfMostRelevantDocuments(
		matchesWithinTermSpread,
		len(criteria.Terms),
		criteria.MaxResults,
	)
	documentMetadata, err := r.documentDirectory.MetadataByHash(ctx, documentHashes)
	if err != nil {
		return Result{}, fmt.Errorf("%w: %w", ErrDocumentDirectory, err)
	}

	report, err := requestedReport.ReportFor(
		ctx,
		r.index,
		r.maxPostingsPerTerm,
		criteria,
		matchesForQueryTerms,
	)
	if err != nil {
		return Result{}, err
	}

	return Result{
		DocumentMetadata: documentMetadata,
		Topics: titletopics.TopicsFromTitles(
			documentMetadata,
			criteria.Terms,
		),
		TotalDocumentsMatchingEveryTerm:   len(matchesWithinTermSpread),
		Duration:                          time.Since(start),
		TotalMatchesPerTerm:               report.TotalMatchesPerTerm,
		PostingsHeldPerTerm:               postingsHeldPerTermFrom(matchesForQueryTerms),
		DocumentsMatchingEachReportedTerm: report.DocumentsMatchingEachReportedTerm,
	}, nil
}

func postingsHeldPerTermFrom(matches map[yacymodel.Hash]termmatch.Match) map[yacymodel.Hash]int {
	held := make(map[yacymodel.Hash]int, len(matches))
	for term, match := range matches {
		held[term] = match.PostingsHeld
	}

	return held
}
