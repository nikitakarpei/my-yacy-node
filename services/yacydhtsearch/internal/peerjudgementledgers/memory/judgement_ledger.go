// Package memory holds the judgements of the peers in this process, for as
// many peers and questions as its capacity allows and for as long as the
// process runs. An instance that restarts starts from no judgement at all.
package memory

import (
	"context"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type peerAndQuestion struct {
	peer     yacymodel.Hash
	question peerjudgements.Question
}

type JudgementLedger struct {
	judgements *expirable.LRU[peerAndQuestion, peerjudgements.RecordedJudgement]
}

func New(capacity int) *JudgementLedger {
	return &JudgementLedger{
		judgements: expirable.NewLRU[peerAndQuestion, peerjudgements.RecordedJudgement](
			capacity, nil, 0,
		),
	}
}

func (l *JudgementLedger) JudgementOf(
	_ context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
) yacymodel.Optional[peerjudgements.RecordedJudgement] {
	judgement, held := l.judgements.Get(peerAndQuestion{peer: peer, question: question})
	if !held {
		return yacymodel.None[peerjudgements.RecordedJudgement]()
	}

	return yacymodel.Some(judgement)
}

func (l *JudgementLedger) HoldJudgement(
	_ context.Context,
	judgement peerjudgements.RecordedJudgement,
) {
	l.judgements.Add(
		peerAndQuestion{peer: judgement.Peer, question: judgement.Question},
		judgement,
	)
}
