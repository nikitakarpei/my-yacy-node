package queryanswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type PostingReplica struct {
	Holder  yacymodel.Hash
	Word    yacymodel.Hash
	Posting yacymodel.RWIPosting
}
