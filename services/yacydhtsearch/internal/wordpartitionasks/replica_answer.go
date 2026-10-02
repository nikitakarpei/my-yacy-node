package wordpartitionasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReplicaAnswer struct {
	Holder                peerdirectory.AskablePeer
	ListedDocuments       []ListedDocument
	AmountOfDocumentsHeld yacymodel.Optional[int]
	Searched              bool
}

func (answer ReplicaAnswer) HasACompleteAbstract() bool {
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeld.Get()
	if !counted {
		return answer.Searched && len(answer.ListedDocuments) == 0
	}

	return amountOfDocumentsHeld <= len(answer.ListedDocuments)
}
