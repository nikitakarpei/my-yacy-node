package searchquery

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type CompoundWord struct {
	Word  string
	Parts []string
}

func (c CompoundWord) Hash() yacymodel.Hash {
	return yacymodel.WordHash(c.Word)
}

func (c CompoundWord) WordHashes() []yacymodel.Hash {
	return hashesOf(c.Parts)
}
