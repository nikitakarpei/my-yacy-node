// Package termdocuments names the documents this node holds for one term, the
// most relevant first. It reads the term in impact order and keeps only the
// documents the search criteria admit, and it stops once it holds as many
// documents as an index abstract of one term covers, or once the request runs
// out of time. An index abstract therefore costs the same reads however many
// results the request asks for.
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

type TermDocuments interface {
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
) TermDocuments {
	return termDocuments{
		postings:                      postings,
		impactOrder:                   impactOrder,
		indexAbstractDocumentsPerTerm: indexAbstractDocumentsPerTerm,
	}
}

type termDocuments struct {
	postings                      rwipostings.PostingIndex
	impactOrder                   rwipostingimpactorder.ImpactOrderQuery
	indexAbstractDocumentsPerTerm int
}

func (t termDocuments) DocumentsHoldingTerm(
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

func (t termDocuments) acceptsDocument(
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
