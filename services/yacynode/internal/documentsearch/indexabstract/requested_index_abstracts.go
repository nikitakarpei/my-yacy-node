package indexabstract

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/termpostings"
)

type RequestedIndexAbstract interface {
	indexAbstracts(
		matchesForQueryTerms map[yacymodel.Hash]termpostings.Match,
		matchesForIndexAbstractTerms map[yacymodel.Hash]termpostings.Match,
	) IndexAbstracts
}

type RequestedIndexAbstracts []RequestedIndexAbstract

func IndexAbstractTermsOf(requested RequestedIndexAbstracts) []yacymodel.Hash {
	var terms []yacymodel.Hash
	for _, requestedAbstract := range requested {
		if abstractsOfTerms, ok := requestedAbstract.(IndexAbstractsOfTerms); ok {
			terms = append(terms, abstractsOfTerms.Terms...)
		}
	}

	return terms
}
