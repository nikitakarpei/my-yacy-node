// Package termdocuments names the documents this node holds for one term, the
// most relevant first. It answers the search pass with the documents an index
// abstract of that term covers.
package termdocuments

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

type TermDocumentQuery interface {
	DocumentsHoldingTerm(
		ctx context.Context,
		tx *vault.Txn,
		term yacymodel.Hash,
		criteria searchcriteria.Criteria,
	) ([]yacymodel.URLHash, error)
}

func New(
	postings rwipostings.PostingIndex,
	impactOrder rwipostingimpactorder.ImpactOrderQuery,
	indexAbstractDocumentsPerTerm int,
) TermDocumentQuery {
	return termDocumentQuery{
		postings:                      postings,
		impactOrder:                   impactOrder,
		indexAbstractDocumentsPerTerm: indexAbstractDocumentsPerTerm,
	}
}

type termDocumentQuery struct {
	postings                      rwipostings.PostingIndex
	impactOrder                   rwipostingimpactorder.ImpactOrderQuery
	indexAbstractDocumentsPerTerm int
}

func (t termDocumentQuery) DocumentsHoldingTerm(
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

			return len(documents) < t.indexAbstractDocumentsPerTerm, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func (t termDocumentQuery) acceptsDocument(
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
