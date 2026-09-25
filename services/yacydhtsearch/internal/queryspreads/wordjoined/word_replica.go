package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordReplica struct {
	answer yacymodel.Optional[peerasks.AnsweredWordAbstractAsk]
}

func (replica wordReplica) hasACompleteAbstract() bool {
	answer, answered := replica.answer.Get()

	return answered && len(answer.Ask.DocumentsToMatch) == 0
}
