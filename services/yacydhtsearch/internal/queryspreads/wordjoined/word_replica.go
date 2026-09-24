package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordReplica struct {
	answer yacymodel.Optional[peerasks.AnsweredSearchDocumentsAsk]
}

func (replica wordReplica) hasACompleteAbstract() bool {
	answer, answered := replica.answer.Get()
	if !answered {
		return false
	}
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()
	if !counted {
		return answer.PeerSearched && len(answer.Abstract) == 0
	}

	return amountOfDocumentsHeld <= len(answer.Abstract)
}
