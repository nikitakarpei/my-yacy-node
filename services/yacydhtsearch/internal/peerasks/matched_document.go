package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type MatchedDocument struct {
	Metadata                yacymodel.URLMetadata
	CountOfAWordTheAskNamed peeranswers.WordCount
}
