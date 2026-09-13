package indexabstract

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type IndexAbstractOfTermNearestToNodePosition struct {
	NodePosition yacymodel.DHTRingPosition
}

func (r IndexAbstractOfTermNearestToNodePosition) coveredTerms(
	queryTerms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) []yacymodel.Hash {
	terms := termsInIndex(queryTerms, amountOfPostingsPerTerm)
	if len(terms) == 0 {
		return nil
	}
	distanceOf := func(term yacymodel.Hash) yacymodel.DHTRingDistance {
		return yacymodel.DHTRingPositionOf(term).DistanceTo(r.NodePosition)
	}

	return []yacymodel.Hash{
		slices.MinFunc(terms, func(a, b yacymodel.Hash) int {
			return cmp.Or(
				cmp.Compare(distanceOf(a), distanceOf(b)),
				yacymodel.CompareInAlphabetOrder(a.String(), b.String()),
			)
		}),
	}
}
