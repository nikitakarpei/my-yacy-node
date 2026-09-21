package queryanswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type PostingReplica struct {
	Holder  yacymodel.Hash
	Word    yacymodel.Optional[yacymodel.Hash]
	Posting yacymodel.RWIPosting
}
