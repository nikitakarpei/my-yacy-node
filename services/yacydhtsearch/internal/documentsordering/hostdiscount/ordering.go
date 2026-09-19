// Package hostdiscount orders the found documents by a relevance that falls by
// half for each document of the same host it already placed above. One host
// thus holds the whole first page only while its further documents stay the
// most relevant ones. Documents of equal discounted relevance keep the order of
// falling relevance, in which documents of equal relevance keep the order the
// spread found them in.
package hostdiscount

import (
	"cmp"
	"math"
	"net/url"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shareOfRelevanceKeptPerPlacedDocumentOfTheSameHost = 0.5
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
	unplacedHostedDocuments := hostedDocumentsOf(documentsOfFallingRelevance)
	amountOfPlacedDocumentsPerHost := map[string]int{}
	placedDocuments := make([]queryanswers.FoundDocument, 0, len(unplacedHostedDocuments))
	for len(unplacedHostedDocuments) > 0 {
		position := positionOfTheHighestDiscountedRelevanceAmong(
			unplacedHostedDocuments, relevancePerDocument, amountOfPlacedDocumentsPerHost,
		)
		placedDocuments = append(placedDocuments, unplacedHostedDocuments[position].document)
		amountOfPlacedDocumentsPerHost[unplacedHostedDocuments[position].host]++
		unplacedHostedDocuments = slices.Delete(unplacedHostedDocuments, position, position+1)
	}

	return placedDocuments
}

type hostedDocument struct {
	document queryanswers.FoundDocument
	host     string
}

func hostedDocumentsOf(foundDocuments []queryanswers.FoundDocument) []hostedDocument {
	hostedDocuments := make([]hostedDocument, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		hostedDocuments = append(hostedDocuments, hostedDocument{
			document: foundDocument,
			host:     hostOf(foundDocument.Address),
		})
	}

	return hostedDocuments
}

func positionOfTheHighestDiscountedRelevanceAmong(
	hostedDocuments []hostedDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedDocumentsPerHost map[string]int,
) int {
	positionOfTheHighestDiscountedRelevance := 0
	highestDiscountedRelevance := math.Inf(-1)
	for position, hostedDocument := range hostedDocuments {
		discountedRelevance := discountedRelevanceOf(
			hostedDocument, relevancePerDocument, amountOfPlacedDocumentsPerHost,
		)
		if discountedRelevance > highestDiscountedRelevance {
			highestDiscountedRelevance = discountedRelevance
			positionOfTheHighestDiscountedRelevance = position
		}
	}

	return positionOfTheHighestDiscountedRelevance
}

func discountedRelevanceOf(
	hostedDocument hostedDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedDocumentsPerHost map[string]int,
) float64 {
	return max(
		relevancePerDocument[hostedDocument.document.Hash],
		leastRelevanceTheDiscountTakesFrom,
	) * math.Pow(
		shareOfRelevanceKeptPerPlacedDocumentOfTheSameHost,
		float64(amountOfPlacedDocumentsPerHost[hostedDocument.host]),
	)
}

func hostOf(address string) string {
	parsedAddress, err := url.Parse(address)
	if err != nil || parsedAddress.Hostname() == "" {
		return address
	}

	return parsedAddress.Hostname()
}
