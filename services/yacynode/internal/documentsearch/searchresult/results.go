// Package searchresult runs one search pass: it joins the query terms over the
// posting index, reads the metadata of the most relevant documents, and adds the
// index abstract the request asked for. The pass reads one snapshot, so the
// documents it returns are the documents the postings chose.
package searchresult

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentmatch"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/titletopics"
)

type DocumentDirectory interface {
	MetadataByHash(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) ([]yacymodel.URLMetadata, error)
}

type Results struct {
	vault             *vault.Vault
	termPostings      termpostings.TermPostings
	documentDirectory DocumentDirectory
}

func New(
	v *vault.Vault,
	postings termpostings.TermPostings,
	documents DocumentDirectory,
) Results {
	return Results{vault: v, termPostings: postings, documentDirectory: documents}
}

type Result struct {
	DocumentMetadata                []yacymodel.URLMetadata
	Topics                          []string
	TotalDocumentsMatchingEveryTerm int
	Duration                        time.Duration
	IndexAbstracts                  indexabstract.IndexAbstracts
	PostingsHeldPerTerm             map[yacymodel.Hash]int
}

var ErrDocumentDirectory = errors.New("document metadata")

func (r Results) ResultFor(
	ctx context.Context,
	criteria searchcriteria.Criteria,
	requestedIndexAbstracts indexabstract.RequestedIndexAbstracts,
) (Result, error) {
	start := time.Now()

	var result Result
	if err := r.vault.View(ctx, func(tx *vault.Txn) error {
		joined, err := r.joinedTerms(ctx, tx, criteria)
		if err != nil {
			return err
		}

		documentMetadata, err := r.documentDirectory.MetadataByHash(tx, joined.documentHashes)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrDocumentDirectory, err)
		}

		matchesForIndexAbstractTerms, err := r.termPostings.MatchesFor(
			ctx,
			tx,
			indexabstract.IndexAbstractTermsOf(requestedIndexAbstracts),
			postingfilter.FilterForReport(criteria),
		)
		if err != nil {
			return err
		}

		abstracts := indexabstract.IndexAbstractsFor(
			requestedIndexAbstracts,
			criteria,
			joined.matchesForQueryTerms,
			matchesForIndexAbstractTerms,
		)

		result = Result{
			DocumentMetadata: documentMetadata,
			Topics: titletopics.TopicsFromTitles(
				documentMetadata,
				criteria.Terms,
			),
			TotalDocumentsMatchingEveryTerm: joined.documentsMatchingEveryTerm,
			IndexAbstracts:                  abstracts,
			PostingsHeldPerTerm: termpostings.PostingsHeldPerTermOf(
				joined.matchesForQueryTerms,
			),
		}

		return nil
	}); err != nil {
		return Result{}, err
	}

	result.Duration = time.Since(start)

	return result, nil
}

type joinedTerms struct {
	matchesForQueryTerms       map[yacymodel.Hash]termpostings.Match
	documentHashes             []yacymodel.URLHash
	documentsMatchingEveryTerm int
}

func (r Results) joinedTerms(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
) (joinedTerms, error) {
	excludedDocuments, err := r.termPostings.DocumentsContaining(ctx, tx, criteria.ExcludedTerms)
	if err != nil {
		return joinedTerms{}, err
	}

	matchesForQueryTerms, err := r.termPostings.MatchesFor(
		ctx,
		tx,
		criteria.Terms,
		postingfilter.FilterForSearch(criteria, excludedDocuments),
	)
	if err != nil {
		return joinedTerms{}, err
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

	return joinedTerms{
		matchesForQueryTerms: matchesForQueryTerms,
		documentHashes: documentmatch.HashesOfMostRelevantDocuments(
			matchesWithinTermSpread,
			len(criteria.Terms),
			criteria.MaxResults,
		),
		documentsMatchingEveryTerm: len(matchesWithinTermSpread),
	}, nil
}
