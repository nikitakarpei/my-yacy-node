package searchendpoint

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func requestedIndexAbstractsFromRequest(
	req yacyproto.SearchRequest,
	nodePosition yacymodel.DHTRingPosition,
) indexabstract.RequestedIndexAbstracts {
	switch req.Abstracts {
	case "":
		return nil
	case yacyproto.SearchAbstractsAuto:
		return requestedIndexAbstractsFromAutoRequest(req, nodePosition)
	default:
		return indexabstract.RequestedIndexAbstracts{
			indexabstract.IndexAbstractsOfTerms{Terms: req.Abstracts.Hashes()},
		}
	}
}

func requestedIndexAbstractsFromAutoRequest(
	req yacyproto.SearchRequest,
	nodePosition yacymodel.DHTRingPosition,
) indexabstract.RequestedIndexAbstracts {
	if len(req.Query) <= 1 || len(req.URLs) != 0 {
		return nil
	}

	return indexabstract.RequestedIndexAbstracts{
		indexabstract.IndexAbstractOfTermWithMostPostings{},
		indexabstract.IndexAbstractOfTermNearestToNodePosition{NodePosition: nodePosition},
	}
}
