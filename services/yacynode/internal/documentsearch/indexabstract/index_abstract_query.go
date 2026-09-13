package indexabstract

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

type IndexAbstractQuery interface {
	IndexAbstractsFor(
		ctx context.Context,
		tx *vault.Txn,
		criteria searchcriteria.Criteria,
		requested RequestedIndexAbstracts,
		amountOfPostingsPerTerm map[yacymodel.Hash]int,
	) (IndexAbstracts, error)
}

func New(
	postings rwipostings.PostingIndex,
	impactOrder rwipostingimpactorder.ImpactOrderQuery,
	documentsPerIndexAbstract int,
) IndexAbstractQuery {
	return indexAbstractQuery{
		postings:                  postings,
		impactOrder:               impactOrder,
		documentsPerIndexAbstract: documentsPerIndexAbstract,
	}
}

type indexAbstractQuery struct {
	postings                  rwipostings.PostingIndex
	impactOrder               rwipostingimpactorder.ImpactOrderQuery
	documentsPerIndexAbstract int
}

func (q indexAbstractQuery) IndexAbstractsFor(
	ctx context.Context,
	tx *vault.Txn,
	criteria searchcriteria.Criteria,
	requested RequestedIndexAbstracts,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) (IndexAbstracts, error) {
	terms := termsCoveredBy(requested, criteria.Terms, amountOfPostingsPerTerm)

	abstracts := make(IndexAbstracts, len(terms))
	for _, term := range terms {
		abstract, err := q.indexAbstractOf(ctx, tx, term, criteria)
		if err != nil {
			return nil, err
		}
		abstracts[term] = abstract
	}

	return abstracts, nil
}

func (q indexAbstractQuery) indexAbstractOf(
	ctx context.Context,
	tx *vault.Txn,
	term yacymodel.Hash,
	criteria searchcriteria.Criteria,
) ([]yacymodel.URLHash, error) {
	filter := postingfilter.FilterForSearch(criteria)

	var documents []yacymodel.URLHash

	err := q.impactOrder.ScanPostingsInImpactOrder(
		tx,
		term,
		func(document yacymodel.URLHash, _ rwipostingimpactorder.Impact) (bool, error) {
			if requestdeadline.RequestHasEnded(ctx) {
				return false, nil
			}
			accepted, err := q.acceptsDocument(tx, term, document, filter)
			if err != nil {
				return false, err
			}
			if accepted {
				documents = append(documents, document)
			}

			return len(documents) < q.documentsPerIndexAbstract, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return documents, nil
}

func (q indexAbstractQuery) acceptsDocument(
	tx *vault.Txn,
	term yacymodel.Hash,
	document yacymodel.URLHash,
	filter postingfilter.Filter,
) (bool, error) {
	posting, found, err := q.postings.PostingOf(tx, term, document)
	if err != nil {
		return false, err
	}

	return found && filter.Accepts(posting), nil
}
