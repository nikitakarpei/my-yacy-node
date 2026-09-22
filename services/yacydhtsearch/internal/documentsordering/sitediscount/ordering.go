// Package sitediscount orders the found documents by a relevance that falls by
// half for each document of the same site it already placed above. One site
// thus holds the whole first page only while its further documents stay the
// most relevant ones. Documents of equal discounted relevance keep the order of
// falling relevance, in which documents of equal relevance keep the order the
// spread found them in.
package sitediscount

import (
	"cmp"
	"container/heap"
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
	positionsPerSite := positionsPerSiteOf(documentsOfFallingRelevance)
	amountOfPlacedDocumentsPerSite := map[string]int{}
	nextDocumentsOfEachSite := &nextDocumentsOfEachSite{}
	for site, positions := range positionsPerSite {
		heap.Push(nextDocumentsOfEachSite, nextDocumentOfSite{
			site:     site,
			position: positions[0],
			discountedRelevance: relevanceOf(
				documentsOfFallingRelevance[positions[0]],
				relevancePerDocument,
			),
		})
	}
	placedDocuments := make([]queryanswers.FoundDocument, 0, len(documentsOfFallingRelevance))
	for nextDocumentsOfEachSite.Len() > 0 {
		placed, _ := heap.Pop(nextDocumentsOfEachSite).(nextDocumentOfSite)
		placedDocuments = append(placedDocuments, documentsOfFallingRelevance[placed.position])
		amountOfPlacedDocumentsPerSite[placed.site]++
		positionsPerSite[placed.site] = positionsPerSite[placed.site][1:]
		if len(positionsPerSite[placed.site]) == 0 {
			continue
		}
		position := positionsPerSite[placed.site][0]
		heap.Push(nextDocumentsOfEachSite, nextDocumentOfSite{
			site:     placed.site,
			position: position,
			discountedRelevance: relevanceOf(
				documentsOfFallingRelevance[position],
				relevancePerDocument,
			) *
				math.Pow(
					shareOfRelevanceKeptPerPlacedDocumentOfTheSameSite,
					float64(amountOfPlacedDocumentsPerSite[placed.site]),
				),
		})
	}

	return placedDocuments
}

func positionsPerSiteOf(foundDocuments []queryanswers.FoundDocument) map[string][]int {
	positionsPerSite := map[string][]int{}
	for position, foundDocument := range foundDocuments {
		site := yacymodel.SiteOf(foundDocument.Address)
		positionsPerSite[site] = append(positionsPerSite[site], position)
	}

	return positionsPerSite
}

func relevanceOf(
	document queryanswers.FoundDocument,
	relevancePerDocument map[yacymodel.URLHash]float64,
) float64 {
	return max(relevancePerDocument[document.Hash], leastRelevanceTheDiscountTakesFrom)
}

type nextDocumentOfSite struct {
	site                string
	position            int
	discountedRelevance float64
}

type nextDocumentsOfEachSite []nextDocumentOfSite

func (next nextDocumentsOfEachSite) Len() int { return len(next) }

func (next nextDocumentsOfEachSite) Less(one, other int) bool {
	if next[one].discountedRelevance != next[other].discountedRelevance {
		return next[one].discountedRelevance > next[other].discountedRelevance
	}

	return next[one].position < next[other].position
}

func (next nextDocumentsOfEachSite) Swap(one, other int) {
	next[one], next[other] = next[other], next[one]
}

func (next *nextDocumentsOfEachSite) Push(item any) {
	nextDocument, _ := item.(nextDocumentOfSite)
	*next = append(*next, nextDocument)
}

func (next *nextDocumentsOfEachSite) Pop() any {
	last := (*next)[len(*next)-1]
	*next = (*next)[:len(*next)-1]

	return last
}
