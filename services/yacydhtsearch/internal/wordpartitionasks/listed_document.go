package wordpartitionasks

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type ListedDocument struct {
	Hash     yacymodel.URLHash
	Metadata yacymodel.Optional[yacymodel.URLMetadata]
	Posting  yacymodel.Optional[yacymodel.RWIPosting]
}
