package indexabstract

import (
	"cmp"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

func indexAbstractOfTermNearestToNodePosition(
	matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
	nodePosition yacymodel.DHTRingPosition,
) IndexAbstracts {
	term, ok := termNearestToNodePosition(matchesForQueryTerms, nodePosition)
	if !ok {
		return nil
	}

	return IndexAbstracts{
		term: documentHashesOf(matchesForQueryTerms[term].PostingPerDocument),
	}
}

func termNearestToNodePosition(
	matches map[yacymodel.Hash]termpostings.Match,
	nodePosition yacymodel.DHTRingPosition,
) (yacymodel.Hash, bool) {
	var (
		nearestTerm      yacymodel.Hash
		shortestDistance yacymodel.DHTRingDistance
		found            bool
	)
	for term, match := range matches {
		if len(match.PostingPerDocument) == 0 {
			continue
		}
		distance := yacymodel.DHTRingPositionOf(term).DistanceTo(nodePosition)
		if !found || distance < shortestDistance ||
			distance == shortestDistance &&
				cmp.Compare(term.String(), nearestTerm.String()) < 0 {
			nearestTerm = term
			shortestDistance = distance
			found = true
		}
	}

	return nearestTerm, found
}
