package peeranswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type DocumentText struct {
	HitsPerQueryWord map[yacymodel.Hash]int
	AmountOfWords    int
	Snippet          string
}
