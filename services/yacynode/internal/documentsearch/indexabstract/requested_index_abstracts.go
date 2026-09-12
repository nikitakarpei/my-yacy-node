package indexabstract

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type RequestedIndexAbstract interface {
	requestedIndexAbstract()
}

type RequestedIndexAbstracts []RequestedIndexAbstract

type IndexAbstractOfTermWithMostPostings struct{}

func (IndexAbstractOfTermWithMostPostings) requestedIndexAbstract() {}

type IndexAbstractOfTermNearestToNodePosition struct {
	NodePosition yacymodel.DHTRingPosition
}

func (IndexAbstractOfTermNearestToNodePosition) requestedIndexAbstract() {}

type IndexAbstractsOfTerms struct {
	Terms []yacymodel.Hash
}

func (IndexAbstractsOfTerms) requestedIndexAbstract() {}

func IndexAbstractTermsOf(requested RequestedIndexAbstracts) []yacymodel.Hash {
	var terms []yacymodel.Hash
	for _, requestedAbstract := range requested {
		if abstractsOfTerms, ok := requestedAbstract.(IndexAbstractsOfTerms); ok {
			terms = append(terms, abstractsOfTerms.Terms...)
		}
	}

	return terms
}
