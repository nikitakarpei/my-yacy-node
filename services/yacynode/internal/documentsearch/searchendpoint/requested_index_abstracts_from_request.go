package searchendpoint

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/indexabstract"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func requestedIndexAbstractsFromRequest(
	req yacyproto.SearchRequest,
) indexabstract.RequestedIndexAbstracts {
	switch req.Abstracts {
	case "":
		return indexabstract.NoIndexAbstracts{}
	case yacyproto.SearchAbstractsAuto:
		return indexabstract.IndexAbstractOfTermWithMostPostings{}
	default:
		return indexabstract.IndexAbstractsOfTerms{Terms: req.Abstracts.Hashes()}
	}
}
