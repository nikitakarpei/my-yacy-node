// Package documentmatch names the documents this node holds for every term of
// one search, and orders them by relevance. It answers the search pass with
// the postings of those documents, how many documents match every term, and
// what ended the read of the index.
package documentmatch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/requestdeadline"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type DocumentMatches struct {
	Postings                           []yacymodel.RWIPosting
	AmountOfDocumentsMatchingEveryTerm int
	IndexReadStop                      IndexReadStop
}

type DocumentMatcher interface {
	MatchesFor(
		ctx context.Context,
		tx *vault.Txn,
		criteria searchcriteria.Criteria,
		amountOfPostingsPerTerm map[yacymodel.Hash]int,
	) (DocumentMatches, error)
}

func New(
	postings rwipostings.PostingIndex,
	impactOrder rwipostingimpactorder.ImpactOrderQuery,
) DocumentMatcher {
	return documentMatcher{postings: postings, impactOrder: impactOrder}
}

type documentMatcher struct {
	postings    rwipostings.PostingIndex
	impactOrder rwipostingimpactorder.ImpactOrderQuery
}

func (m documentMatcher) MatchesFor(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) (DocumentMatches, error) {
	if !everyTermHasPostings(criteria.Terms, amountOfPostingsPerTerm) {
		return DocumentMatches{}, nil
	}
	terms := termsByRarityOf(criteria.Terms, amountOfPostingsPerTerm)
	relevanceBoundOfOtherTerms, err := m.relevanceBoundOfOtherTerms(tx, terms)
	if err != nil {
		return DocumentMatches{}, err
	}

	ranking := rankingFor(criteria, terms, relevanceBoundOfOtherTerms)
	indexReadStop, err := m.readRarestTerm(ctx, tx, criteria, &ranking)
	if err != nil {
		return DocumentMatches{}, err
	}

	return DocumentMatches{
		Postings:                           ranking.postingsInRelevanceOrder(),
		AmountOfDocumentsMatchingEveryTerm: ranking.amountOfDocumentsMatchingEveryTerm,
		IndexReadStop:                      indexReadStop,
	}, nil
}

func everyTermHasPostings(
	terms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) bool {
	if len(terms) == 0 {
		return false
	}
	for _, term := range terms {
		if amountOfPostingsPerTerm[term] <= 0 {
			return false
		}
	}

	return true
}

func (m documentMatcher) relevanceBoundOfOtherTerms(
	tx *vault.Txn,
	terms termsByRarity,
) (float64, error) {
	relevanceBound := 0.0
	for _, term := range terms.otherTerms {
		largestImpact, found, err := m.impactOrder.LargestImpactOf(tx, term)
		if err != nil {
			return 0, err
		}
		if !found {
			continue
		}
		relevanceBound += terms.rarityOf(term) * float64(largestImpact)
	}

	return relevanceBound, nil
}

func (m documentMatcher) readRarestTerm(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	ranking *ranking,
) (IndexReadStop, error) {
	indexReadStop := IndexReadStoppedAtEndOfTerm
	filter := postingfilter.FilterForSearch(criteria)
	err := m.impactOrder.ScanPostingsInImpactOrder(
		tx,
		ranking.rarestTerm(),
		func(
			document yacymodel.URLHash,
			impactOfNextUnreadPosting rwipostingimpactorder.Impact,
		) (bool, error) {
			if requestdeadline.RequestHasEnded(ctx) {
				indexReadStop = IndexReadStoppedAtDeadline

				return false, nil
			}
			if ranking.hasReachedRelevanceBoundAt(impactOfNextUnreadPosting) {
				indexReadStop = IndexReadStoppedAtRelevanceBound

				return false, nil
			}
			if err := m.rankDocument(tx, criteria, filter, document, ranking); err != nil {
				return false, err
			}

			return true, nil
		},
	)

	return indexReadStop, err
}

func (m documentMatcher) rankDocument(
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	filter postingfilter.Filter,
	document yacymodel.URLHash,
	ranking *ranking,
) error {
	postings, holdsEveryTerm, err := m.postingsOfEveryTerm(tx, criteria.Terms, filter, document)
	if err != nil {
		return err
	}
	if !holdsEveryTerm {
		return nil
	}
	holdsAnExcludedTerm, err := m.documentHoldsAnyOf(tx, criteria.ExcludedTerms, document)
	if err != nil {
		return err
	}
	if holdsAnExcludedTerm {
		return nil
	}
	ranking.rankPostingsOfDocument(postings)

	return nil
}

func (m documentMatcher) postingsOfEveryTerm(
	tx *vault.Txn,
	terms []yacymodel.Hash,
	filter postingfilter.Filter,
	document yacymodel.URLHash,
) ([]yacymodel.RWIPosting, bool, error) {
	postings := make([]yacymodel.RWIPosting, 0, len(terms))
	for _, term := range terms {
		posting, found, err := m.postings.PostingOf(tx, term, document)
		if err != nil {
			return nil, false, err
		}
		if !found || !filter.Accepts(posting) {
			return nil, false, nil
		}
		postings = append(postings, posting)
	}

	return postings, true, nil
}

func (m documentMatcher) documentHoldsAnyOf(
	tx *vault.Txn,
	terms []yacymodel.Hash,
	document yacymodel.URLHash,
) (bool, error) {
	for _, term := range terms {
		_, found, err := m.postings.PostingOf(tx, term, document)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}

	return false, nil
}
