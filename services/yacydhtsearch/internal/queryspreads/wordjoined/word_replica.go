package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordReplica struct {
	peer   peerdirectory.AskablePeer
	answer yacymodel.Optional[peerasks.AnsweredMatchedAndHeldDocumentsAsk]
}

func (replica wordReplica) isFullyListed() bool {
	answer, answered := replica.answer.Get()
	if !answered {
		return false
	}
	amountOfDocumentsHeld, counted := answer.AmountOfDocumentsHeldForTheWord.Get()

	return counted && amountOfDocumentsHeld <= len(answer.DocumentsListedForTheWord)
}

func (replica wordReplica) versionClaimed() string {
	answer, answered := replica.answer.Get()
	if !answered {
		return ""
	}

	return answer.PeerVersion
}
