package indexabstract

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type RequestedIndexAbstracts interface {
	requestedIndexAbstracts()
}

type NoIndexAbstracts struct{}

func (NoIndexAbstracts) requestedIndexAbstracts() {}

type IndexAbstractOfTermWithMostPostings struct{}

func (IndexAbstractOfTermWithMostPostings) requestedIndexAbstracts() {}

type IndexAbstractsOfTerms struct {
	Terms []yacymodel.Hash
}

func (IndexAbstractsOfTerms) requestedIndexAbstracts() {}

func IndexAbstractTermsOf(requested RequestedIndexAbstracts) []yacymodel.Hash {
	if abstract, ok := requested.(IndexAbstractsOfTerms); ok {
		return abstract.Terms
	}

	return nil
}
