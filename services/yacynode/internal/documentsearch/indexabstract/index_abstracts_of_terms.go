package indexabstract

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

type IndexAbstractsOfTerms struct {
	Terms []yacymodel.Hash
}

func (r IndexAbstractsOfTerms) indexAbstracts(
	_ map[yacymodel.Hash]termpostings.Match,
	matchesForIndexAbstractTerms map[yacymodel.Hash]termpostings.Match,
) IndexAbstracts {
	abstracts := make(IndexAbstracts, len(r.Terms))
	for _, term := range r.Terms {
		abstracts[term] = documentHashesOf(matchesForIndexAbstractTerms[term].PostingPerDocument)
	}

	return abstracts
}
