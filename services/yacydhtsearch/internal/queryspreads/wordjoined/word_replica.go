package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordReplica struct {
	listedDocuments  []wordpartitionasks.ListedDocument
	documentsToMatch []yacymodel.URLHash
}

func (replica wordReplica) hasACompleteAbstract() bool {
	return len(replica.documentsToMatch) == 0
}
