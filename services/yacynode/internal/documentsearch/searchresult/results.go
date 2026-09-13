// Package searchresult owns one search pass over the index of this node. It
// answers the search endpoint with the matched documents and their metadata,
// the topics of their titles, the index abstracts the request asked for, and
// how many postings this node holds for each query term.
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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termdocuments"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/titletopics"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingamount"
)

type DocumentDirectory interface {
	MetadataPerHash(
		tx *vault.Txn,
		hashes []yacymodel.URLHash,
	) (map[yacymodel.URLHash]yacymodel.URLMetadata, error)
}

type Results struct {
	vault             *vault.Vault
	documentMatcher   documentmatch.DocumentMatcher
	termDocumentQuery termdocuments.TermDocumentQuery
	postingAmounts    rwipostingamount.PostingAmountQuery
	documentDirectory DocumentDirectory
}

func New(
	v *vault.Vault,
	documentMatcher documentmatch.DocumentMatcher,
	termDocumentQuery termdocuments.TermDocumentQuery,
	postingAmounts rwipostingamount.PostingAmountQuery,
	documentDirectory DocumentDirectory,
) Results {
	return Results{
		vault:             v,
		documentMatcher:   documentMatcher,
		termDocumentQuery: termDocumentQuery,
		postingAmounts:    postingAmounts,
		documentDirectory: documentDirectory,
	}
}

type MatchedDocument struct {
	Metadata yacymodel.URLMetadata
	Posting  yacymodel.RWIPosting
}

type Result struct {
	MatchedDocuments                []MatchedDocument
	Topics                          []string
	TotalDocumentsMatchingEveryTerm int
	Duration                        time.Duration
	IndexAbstracts                  indexabstract.IndexAbstracts
	AmountOfPostingsPerTerm         map[yacymodel.Hash]int
	IndexReadStop                   documentmatch.IndexReadStop
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
		found, err := r.resultIn(ctx, tx, criteria, requestedIndexAbstracts)
		result = found

		return err
	}); err != nil {
		return Result{}, err
	}

	result.Duration = time.Since(start)

	return result, nil
}

func (r Results) resultIn(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	requestedIndexAbstracts indexabstract.RequestedIndexAbstracts,
) (Result, error) {
	amountOfPostingsPerTerm, err := r.amountOfPostingsPerTerm(tx, criteria.Terms)
	if err != nil {
		return Result{}, err
	}

	documentMatches, err := r.documentMatcher.MatchesFor(
		ctx, tx, criteria, amountOfPostingsPerTerm,
	)
	if err != nil {
		return Result{}, err
	}

	matchedDocuments, err := r.matchedDocuments(tx, documentMatches.Postings)
	if err != nil {
		return Result{}, err
	}

	abstracts, err := r.indexAbstracts(
		ctx, tx, criteria, requestedIndexAbstracts, amountOfPostingsPerTerm,
	)
	if err != nil {
		return Result{}, err
	}

	return Result{
		MatchedDocuments: matchedDocuments,
		Topics: titletopics.TopicsFromTitles(
			documentTitlesOf(matchedDocuments),
			criteria.Terms,
		),
		TotalDocumentsMatchingEveryTerm: documentMatches.AmountOfDocumentsMatchingEveryTerm,
		IndexAbstracts:                  abstracts,
		AmountOfPostingsPerTerm:         amountOfPostingsPerTerm,
		IndexReadStop:                   documentMatches.IndexReadStop,
	}, nil
}

func (r Results) amountOfPostingsPerTerm(
	tx *vault.Txn,
	terms []yacymodel.Hash,
) (map[yacymodel.Hash]int, error) {
	amountPerTerm := make(map[yacymodel.Hash]int, len(terms))
	for _, term := range terms {
		amountOfPostings, err := r.postingAmounts.AmountOfPostingsOf(tx, term)
		if err != nil {
			return nil, err
		}
		amountPerTerm[term] = amountOfPostings
	}

	return amountPerTerm, nil
}

func (r Results) matchedDocuments(
	tx *vault.Txn,
	postings []yacymodel.RWIPosting,
) ([]MatchedDocument, error) {
	metadataPerHash, err := r.documentDirectory.MetadataPerHash(tx, documentHashesOf(postings))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDocumentDirectory, err)
	}

	matched := make([]MatchedDocument, 0, len(postings))
	for _, posting := range postings {
		metadata, ok := metadataPerHash[posting.URLHash]
		if !ok {
			continue
		}
		matched = append(matched, MatchedDocument{Metadata: metadata, Posting: posting})
	}

	return matched, nil
}

func documentHashesOf(postings []yacymodel.RWIPosting) []yacymodel.URLHash {
	hashes := make([]yacymodel.URLHash, 0, len(postings))
	for _, posting := range postings {
		hashes = append(hashes, posting.URLHash)
	}

	return hashes
}

func (r Results) indexAbstracts(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	requestedIndexAbstracts indexabstract.RequestedIndexAbstracts,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) (indexabstract.IndexAbstracts, error) {
	terms := indexabstract.IndexAbstractTermsOf(
		requestedIndexAbstracts,
		criteria.Terms,
		amountOfPostingsPerTerm,
	)

	documentsPerTerm := make(map[yacymodel.Hash][]yacymodel.URLHash, len(terms))
	for _, term := range terms {
		documents, err := r.termDocumentQuery.DocumentsHoldingTerm(ctx, tx, term, criteria)
		if err != nil {
			return nil, err
		}
		documentsPerTerm[term] = documents
	}

	return indexabstract.IndexAbstractsOf(terms, documentsPerTerm), nil
}

func documentTitlesOf(matched []MatchedDocument) []string {
	titles := make([]string, 0, len(matched))
	for _, document := range matched {
		titles = append(titles, document.Metadata.Title)
	}

	return titles
}
