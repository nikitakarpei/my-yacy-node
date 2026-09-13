// Package documentmatch names the documents this node holds for every term of
// one search, the most relevant first. It answers the search pass with the
// postings of those documents, how many documents match every term, and what
// ended the read of the index.
package documentmatch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/documentshortlist"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/requestdeadline"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type DocumentMatches struct {
	JoinedPostings                     []yacymodel.RWIPosting
	AmountOfDocumentsMatchingEveryTerm int
	IndexReadStopReason                IndexReadStopReason
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

type indexRead struct {
	terms                              searchrelevance.SearchTerms
	relevanceBound                     searchrelevance.RelevanceBound
	shortlist                          *documentshortlist.Shortlist
	amountOfDocumentsMatchingEveryTerm int
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
	terms := searchrelevance.SearchTermsOf(criteria, amountOfPostingsPerTerm)
	largestImpactPerOtherTerm, err := m.largestImpactPerOtherTerm(tx, terms)
	if err != nil {
		return DocumentMatches{}, err
	}

	rarestTermRead := indexRead{
		terms:          terms,
		relevanceBound: searchrelevance.RelevanceBoundOf(terms, largestImpactPerOtherTerm),
		shortlist:      documentshortlist.New(criteria.MaxResults),
	}
	indexReadStopReason, err := m.readRarestTerm(ctx, tx, criteria, &rarestTermRead)
	if err != nil {
		return DocumentMatches{}, err
	}

	return DocumentMatches{
		JoinedPostings: joinedPostingsOf(
			rarestTermRead.shortlist.InRelevanceOrder(),
		),
		AmountOfDocumentsMatchingEveryTerm: rarestTermRead.amountOfDocumentsMatchingEveryTerm,
		IndexReadStopReason:                indexReadStopReason,
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

func (m documentMatcher) largestImpactPerOtherTerm(
	tx *vault.Txn,
	terms searchrelevance.SearchTerms,
) (map[yacymodel.Hash]rwipostingimpactorder.Impact, error) {
	largestImpactPerOtherTerm := make(
		map[yacymodel.Hash]rwipostingimpactorder.Impact,
		len(terms.OtherTerms()),
	)
	for _, term := range terms.OtherTerms() {
		largestImpact, found, err := m.impactOrder.LargestImpactOf(tx, term)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		largestImpactPerOtherTerm[term] = largestImpact
	}

	return largestImpactPerOtherTerm, nil
}

func (m documentMatcher) readRarestTerm(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	rarestTermRead *indexRead,
) (IndexReadStopReason, error) {
	indexReadStopReason := IndexReadStoppedAtEndOfTerm
	filter := postingfilter.FilterForSearch(criteria)
	err := m.impactOrder.ScanPostingsInImpactOrder(
		tx,
		rarestTermRead.terms.RarestTerm(),
		func(
			document yacymodel.URLHash,
			impactOfNextUnreadPosting rwipostingimpactorder.Impact,
		) (bool, error) {
			if requestdeadline.RequestHasEnded(ctx) {
				indexReadStopReason = IndexReadStoppedAtDeadline

				return false, nil
			}
			if rarestTermRead.shortlist.IsFull() &&
				rarestTermRead.shortlist.LowestRelevance() >=
					rarestTermRead.relevanceBound.LargestRelevanceReachableFrom(
						impactOfNextUnreadPosting,
					) {
				indexReadStopReason = IndexReadStoppedAtRelevanceBound

				return false, nil
			}
			if err := m.placeDocument(tx, criteria, filter, document, rarestTermRead); err != nil {
				return false, err
			}

			return true, nil
		},
	)

	return indexReadStopReason, err
}

func (m documentMatcher) placeDocument(
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	filter postingfilter.Filter,
	document yacymodel.URLHash,
	rarestTermRead *indexRead,
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
	termSpread := termSpreadOf(postings)
	if criteria.MaxTermSpread > 0 && termSpread > criteria.MaxTermSpread {
		return nil
	}
	rarestTermRead.amountOfDocumentsMatchingEveryTerm++
	rarestTermRead.shortlist.Place(documentshortlist.RankedDocument{
		JoinedPosting: joinedPostingOf(postings),
		Relevance:     searchrelevance.RelevanceOf(postings, rarestTermRead.terms),
		TermSpread:    termSpread,
	})

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

func joinedPostingsOf(documents []documentshortlist.RankedDocument) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(documents))
	for _, document := range documents {
		postings = append(postings, document.JoinedPosting)
	}

	return postings
}
