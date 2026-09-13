package indexabstract

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

type IndexAbstractOfTermNearestToNodePosition struct {
	NodePosition yacymodel.DHTRingPosition
}

func (r IndexAbstractOfTermNearestToNodePosition) indexAbstracts(
	matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
	_ map[yacymodel.Hash]termpostings.Match,
) IndexAbstracts {
	term, ok := termNearestToNodePositionOf(matchesForQueryTerms, r.NodePosition)
	if !ok {
		return nil
	}

	return IndexAbstracts{
		term: documentHashesOf(matchesForQueryTerms[term].PostingPerDocument),
	}
}

func termNearestToNodePositionOf(
	matches map[yacymodel.Hash]termpostings.Match,
	nodePosition yacymodel.DHTRingPosition,
) (yacymodel.Hash, bool) {
	terms := termsWithDocumentsOf(matches)
	if len(terms) == 0 {
		return yacymodel.Hash{}, false
	}
	distanceOf := func(term yacymodel.Hash) yacymodel.DHTRingDistance {
		return yacymodel.DHTRingPositionOf(term).DistanceTo(nodePosition)
	}

	return slices.MinFunc(terms, func(a, b yacymodel.Hash) int {
		return cmp.Or(
			cmp.Compare(distanceOf(a), distanceOf(b)),
			cmp.Compare(a.String(), b.String()),
		)
	}), true
}
