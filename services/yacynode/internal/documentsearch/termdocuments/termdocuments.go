// Package termdocuments names the documents this node holds for one term, the
// most relevant first. It reads the term in impact order and keeps only the
// documents the search criteria admit, and it stops once it holds as many
// documents as the request asks results for, or once the request runs out of
// time. An index abstract of a term therefore costs no more reads than the
// answer the same request carries.
package termdocuments

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiimpactorder"
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
	impactOrder rwiimpactorder.ImpactOrderQuery,
) TermDocuments {
	return termDocuments{postings: postings, impactOrder: impactOrder}
}

type termDocuments struct {
	postings    rwipostings.PostingIndex
	impactOrder rwiimpactorder.ImpactOrderQuery
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
		func(document yacymodel.URLHash, _ rwiimpactorder.Impact) (bool, error) {
			if requestHasEnded(ctx) {
				return false, nil
			}
			accepted, err := t.acceptsDocument(tx, term, document, filter)
			if err != nil {
				return false, err
			}
			if accepted {
				documents = append(documents, document)
			}

			return criteria.MaxResults <= 0 || len(documents) < criteria.MaxResults, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func requestHasEnded(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
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
