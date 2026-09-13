// Package termmatch names the documents this node holds for one term that the
// search criteria admit, the most relevant first and up to a cap. It answers
// the search pass with them for the index abstracts of that term.
package termmatch

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

type TermMatcher interface {
	MatchesFor(
		ctx context.Context,
		tx *vault.Txn,
		term yacymodel.Hash,
		criteria searchcriteria.Criteria,
	) ([]yacymodel.URLHash, error)
}

func New(
	postings rwipostings.PostingIndex,
	impactOrder rwipostingimpactorder.ImpactOrderQuery,
	mostRelevantDocumentsPerTerm int,
) TermMatcher {
	return termMatcher{
		postings:                     postings,
		impactOrder:                  impactOrder,
		mostRelevantDocumentsPerTerm: mostRelevantDocumentsPerTerm,
	}
}

type termMatcher struct {
	postings                     rwipostings.PostingIndex
	impactOrder                  rwipostingimpactorder.ImpactOrderQuery
	mostRelevantDocumentsPerTerm int
}

func (t termMatcher) MatchesFor(
	ctx context.Context,
	tx *vault.Txn,
	term yacymodel.Hash,
	criteria searchcriteria.Criteria,
) ([]yacymodel.URLHash, error) {
	filter := postingfilter.FilterForSearch(criteria)

	var documents []yacymodel.URLHash

	err := t.impactOrder.ScanPostingsInImpactOrder(
		tx,
		term,
		func(document yacymodel.URLHash, _ rwipostingimpactorder.Impact) (bool, error) {
			if requestdeadline.RequestHasEnded(ctx) {
				return false, nil
			}
			accepted, err := t.acceptsDocument(tx, term, document, filter)
			if err != nil {
				return false, err
			}
			if accepted {
				documents = append(documents, document)
			}

			return len(documents) < t.mostRelevantDocumentsPerTerm, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func (t termMatcher) acceptsDocument(
	tx *vault.Txn,
	term yacymodel.Hash,
	document yacymodel.URLHash,
	filter postingfilter.Filter,
) (bool, error) {
	posting, found, err := t.postings.PostingOf(tx, term, document)
	if err != nil {
		return false, err
	}

	return found && filter.Accepts(posting), nil
}
