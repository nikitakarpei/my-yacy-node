// Package sitediscount orders the found documents by a relevance that falls by
// half for each document of the same site it already placed above. One site
// thus holds the whole first page only while its further documents stay the
// most relevant ones. Documents of equal discounted relevance keep the order of
// falling relevance, in which documents of equal relevance keep the order the
// spread found them in.
package sitediscount

import (
	"cmp"
	"math"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shareOfRelevanceKeptPerPlacedDocumentOfTheSameSite = 0.5
	leastRelevanceTheDiscountTakesFrom                 = 0.0
)

type DocumentRelevance interface {
	RelevancePerDocumentOf(answers queryanswers.AnsweredQuery) map[yacymodel.URLHash]float64
}

type Ordering struct {
	documentRelevance DocumentRelevance
}

func New(documentRelevance DocumentRelevance) Ordering {
	return Ordering{documentRelevance: documentRelevance}
}

func (ordering Ordering) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	relevancePerDocument := ordering.documentRelevance.RelevancePerDocumentOf(answers)

	return documentsInFallingOrderOfDiscountedRelevance(
		documentsInFallingOrderOfRelevance(
			slices.Clone(answers.FoundDocuments), relevancePerDocument,
		),
		relevancePerDocument,
	)
}

func documentsInFallingOrderOfRelevance(
	foundDocuments []queryanswers.FoundDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []queryanswers.FoundDocument {
	slices.SortStableFunc(foundDocuments, func(one, other queryanswers.FoundDocument) int {
		return cmp.Compare(
			relevancePerDocument[other.Hash], relevancePerDocument[one.Hash],
		)
	})

	return foundDocuments
}

func documentsInFallingOrderOfDiscountedRelevance(
	documentsOfFallingRelevance []queryanswers.FoundDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []queryanswers.FoundDocument {
	unplacedSitedDocuments := sitedDocumentsOf(documentsOfFallingRelevance)
	amountOfPlacedDocumentsPerSite := map[string]int{}
	placedDocuments := make([]queryanswers.FoundDocument, 0, len(unplacedSitedDocuments))
	for len(unplacedSitedDocuments) > 0 {
		position := positionOfTheHighestDiscountedRelevanceAmong(
			unplacedSitedDocuments, relevancePerDocument, amountOfPlacedDocumentsPerSite,
		)
		placedDocuments = append(placedDocuments, unplacedSitedDocuments[position].document)
		amountOfPlacedDocumentsPerSite[unplacedSitedDocuments[position].site]++
		unplacedSitedDocuments = slices.Delete(unplacedSitedDocuments, position, position+1)
	}

	return placedDocuments
}

type sitedDocument struct {
	document queryanswers.FoundDocument
	site     string
}

func sitedDocumentsOf(foundDocuments []queryanswers.FoundDocument) []sitedDocument {
	sitedDocuments := make([]sitedDocument, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		sitedDocuments = append(sitedDocuments, sitedDocument{
			document: foundDocument,
			site:     yacymodel.SiteOf(foundDocument.Address),
		})
	}

	return sitedDocuments
}

func positionOfTheHighestDiscountedRelevanceAmong(
	sitedDocuments []sitedDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedDocumentsPerSite map[string]int,
) int {
	positionOfTheHighestDiscountedRelevance := 0
	highestDiscountedRelevance := math.Inf(-1)
	for position, sitedDocument := range sitedDocuments {
		discountedRelevance := discountedRelevanceOf(
			sitedDocument, relevancePerDocument, amountOfPlacedDocumentsPerSite,
		)
		if discountedRelevance > highestDiscountedRelevance {
			highestDiscountedRelevance = discountedRelevance
			positionOfTheHighestDiscountedRelevance = position
		}
	}

	return positionOfTheHighestDiscountedRelevance
}

func discountedRelevanceOf(
	sitedDocument sitedDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedDocumentsPerSite map[string]int,
) float64 {
	return max(
		relevancePerDocument[sitedDocument.document.Hash],
		leastRelevanceTheDiscountTakesFrom,
	) * math.Pow(
		shareOfRelevanceKeptPerPlacedDocumentOfTheSameSite,
		float64(amountOfPlacedDocumentsPerSite[sitedDocument.site]),
	)
}
