package queryanswers

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type MetadataReplica struct {
	Holder   yacymodel.Hash
	Metadata yacymodel.URLMetadata
}
