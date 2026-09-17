package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type MatchedDocument struct {
	Metadata                yacymodel.URLMetadata
	CountOfAWordTheAskNamed queryanswers.WordCount
}
