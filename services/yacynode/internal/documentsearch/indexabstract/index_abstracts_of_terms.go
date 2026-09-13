package indexabstract

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type IndexAbstractsOfTerms struct {
	Terms []yacymodel.Hash
}

func (r IndexAbstractsOfTerms) indexAbstractTerms(
	_ []yacymodel.Hash,
	_ map[yacymodel.Hash]int,
) []yacymodel.Hash {
	return r.Terms
}
