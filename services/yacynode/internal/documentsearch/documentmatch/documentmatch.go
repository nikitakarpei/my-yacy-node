// Package documentmatch names the documents that hold every word of one search
// and orders them by relevance. It starts from the word with the fewest
// postings and reads that word in impact order, so the most relevant documents
// of that word come first. For every document it reads, it looks the other
// search words, the filters and the excluded words up by key, and it adds up
// one relevance from the impact of each posting, weighted by how rare its word
// is in this node. It stops once no document it has not read can still enter
// the answer, and it stops when the request runs out of time.
//
// The posting it carries back for one document merges the postings of every
// search word the way YaCy's own join does, the earliest positions and the
// largest counts, so the posting this node hands a peer is the posting that
// peer would have built for itself. Documents of equal relevance are ordered by
// term spread, the average gap between the text positions of the search words,
// and then by document hash.
package documentmatch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type MostRelevantDocuments struct {
	Postings                 []yacymodel.RWIPosting
	AmountOfMatchedDocuments int
	IndexReadStop            IndexReadStop
}

type Matches interface {
	MostRelevantDocumentsFor(
		ctx context.Context,
		tx *vault.Txn,
		criteria searchcriteria.Criteria,
		amountOfPostingsPerTerm map[yacymodel.Hash]int,
	) (MostRelevantDocuments, error)
}

func New(
	postings rwipostings.PostingIndex,
	impactOrder rwiimpactorder.ImpactOrderQuery,
) Matches {
	return indexMatches{postings: postings, impactOrder: impactOrder}
}

type indexMatches struct {
	postings    rwipostings.PostingIndex
	impactOrder rwiimpactorder.ImpactOrderQuery
}

func (m indexMatches) MostRelevantDocumentsFor(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) (MostRelevantDocuments, error) {
	rarity := wordRarityOf(criteria.Terms, amountOfPostingsPerTerm)
	if !rarity.everyWordIsHeld {
		return MostRelevantDocuments{}, nil
	}
	largestRelevanceBesideRarestWord, err := m.largestRelevanceBesideRarestWord(tx, rarity)
	if err != nil {
		return MostRelevantDocuments{}, err
	}

	ranking := rankingFor(criteria, rarity, largestRelevanceBesideRarestWord)
	indexReadStop, err := m.readRarestWord(ctx, tx, criteria, &ranking)
	if err != nil {
		return MostRelevantDocuments{}, err
	}

	return MostRelevantDocuments{
		Postings:                 ranking.postingsInRelevanceOrder(),
		AmountOfMatchedDocuments: ranking.amountOfMatchedDocuments,
		IndexReadStop:            indexReadStop,
	}, nil
}

func (m indexMatches) largestRelevanceBesideRarestWord(
	tx *vault.Txn,
	rarity wordRarity,
) (float64, error) {
	largestRelevance := 0.0
	for position, word := range rarity.words {
		if position == rarity.rarestWordPosition {
			continue
		}
		largestImpact, found, err := m.impactOrder.LargestImpactOf(tx, word)
		if err != nil {
			return 0, err
		}
		if !found {
			continue
		}
		largestRelevance += rarity.rarityOf(word) * float64(largestImpact)
	}

	return largestRelevance, nil
}

func (m indexMatches) readRarestWord(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	ranking *ranking,
) (IndexReadStop, error) {
	indexReadStop := StoppedAtEveryPosting
	filter := postingfilter.FilterForSearch(criteria)
	err := m.impactOrder.ScanPostingsInImpactOrder(
		tx,
		ranking.rarestWord(),
		func(document yacymodel.URLHash, impact rwiimpactorder.Impact) (bool, error) {
			if requestHasEnded(ctx) {
				indexReadStop = StoppedAtDeadline

				return false, nil
			}
			if ranking.noUnreadDocumentCanEnterTheAnswer(impact) {
				indexReadStop = StoppedAtRelevanceBound

				return false, nil
			}
			if err := m.considerDocument(tx, criteria, filter, document, ranking); err != nil {
				return false, err
			}

			return true, nil
		},
	)

	return indexReadStop, err
}

func requestHasEnded(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func (m indexMatches) considerDocument(
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
	holdsAnExcludedTerm, err := m.holdsAnyTerm(tx, criteria.ExcludedTerms, document)
	if err != nil {
		return err
	}
	if holdsAnExcludedTerm {
		return nil
	}
	ranking.consider(postings)

	return nil
}

func (m indexMatches) postingsOfEveryTerm(
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

func (m indexMatches) holdsAnyTerm(
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
