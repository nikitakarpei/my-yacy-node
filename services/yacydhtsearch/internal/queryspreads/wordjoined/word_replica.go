package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"

type wordReplica struct {
	answer wordpartitionasks.ReplicaAnswer
}

func (replica wordReplica) hasACompleteAbstract() bool {
	amountOfDocumentsHeld, counted := replica.answer.AmountOfDocumentsHeld.Get()
	if !counted {
		return replica.answer.Searched && len(replica.answer.ListedDocuments) == 0
	}

	return amountOfDocumentsHeld <= len(replica.answer.ListedDocuments)
}
