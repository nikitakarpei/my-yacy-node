package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type MatchedDocument struct {
	Metadata yacymodel.URLMetadata
	Posting  yacymodel.Optional[yacymodel.RWIPosting]
}
