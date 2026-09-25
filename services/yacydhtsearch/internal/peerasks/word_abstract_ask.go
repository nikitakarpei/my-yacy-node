package peerasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type WordAbstractAsk struct {
	Peer             peerdirectory.AskablePeer
	Partition        uint
	Word             yacymodel.Hash
	ExcludedWords    []yacymodel.Hash
	Language         string
	DocumentsToMatch []yacymodel.URLHash
}

type WordAbstractAskOutcome = AskOutcome[WordAbstractAsk, AnsweredWordAbstractAsk]

type WordAbstractAskOutcomes = AskOutcomes[WordAbstractAsk, AnsweredWordAbstractAsk]

type AnsweredWordAbstractAsk struct {
	Ask      WordAbstractAsk
	Abstract []yacymodel.URLHash
}
