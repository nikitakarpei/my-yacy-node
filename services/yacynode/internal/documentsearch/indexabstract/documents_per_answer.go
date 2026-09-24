package indexabstract

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type DocumentsPerAnswer int

func (d DocumentsPerAnswer) sharedAmong(terms []yacymodel.Hash) int {
	if len(terms) == 0 {
		return 0
	}

	return int(d) / len(terms)
}
