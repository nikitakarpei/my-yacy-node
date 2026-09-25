// Package welllinkedhostrelevance lowers the relevance of each found document
// whose host is not among the well-linked hosts, so that a document of a host
// that few others link to ranks below an as relevant document of a host that
// many others link to. A relevance of zero or less stays as it is.
package welllinkedhostrelevance

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const shareOfRelevanceKeptOutsideTheWellLinkedHosts = 0.9

type DocumentRelevance interface {
	RelevancePerDocumentOf(
		ctx context.Context,
		answers queryanswers.AnsweredQuery,
	) map[yacymodel.URLHash]float64
}

type WellLinkedHosts interface {
	HoldsHostOf(address string) bool
}

type Relevance struct {
	documentRelevance DocumentRelevance
	wellLinkedHosts   WellLinkedHosts
	observer          RelevanceObserver
}

func New(
	documentRelevance DocumentRelevance,
	wellLinkedHosts WellLinkedHosts,
	observer RelevanceObserver,
) Relevance {
	return Relevance{
		documentRelevance: documentRelevance,
		wellLinkedHosts:   wellLinkedHosts,
		observer:          observer,
	}
}

func (relevance Relevance) RelevancePerDocumentOf(
	ctx context.Context,
	answers queryanswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	relevancePerDocument := relevance.documentRelevance.RelevancePerDocumentOf(ctx, answers)
	amountOfDemotedDocuments := 0
	for _, document := range answers.FoundDocuments {
		if relevancePerDocument[document.Hash] <= 0 ||
			relevance.wellLinkedHosts.HoldsHostOf(document.Address) {
			continue
		}
		relevancePerDocument[document.Hash] *= shareOfRelevanceKeptOutsideTheWellLinkedHosts
		amountOfDemotedDocuments++
	}
	relevance.observer.DocumentsDemoted(ctx, amountOfDemotedDocuments)

	return relevancePerDocument
}
