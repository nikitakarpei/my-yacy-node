// Package memory holds the judgements of the peers in this process, for as
// many peers and forms as its capacity allows and for as long as the process
// runs. An instance that restarts starts from no judgement at all.
package memory

import (
	"context"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type formAndPeer struct {
	form peerjudgements.Form
	peer yacymodel.Hash
}

type JudgementLedger struct {
	judgements *expirable.LRU[formAndPeer, peerjudgements.RecordedJudgement]
}

func New(capacity int) *JudgementLedger {
	return &JudgementLedger{
		judgements: expirable.NewLRU[formAndPeer, peerjudgements.RecordedJudgement](
			capacity, nil, 0,
		),
	}
}

func (l *JudgementLedger) JudgementOf(
	_ context.Context,
	form peerjudgements.Form,
	peer yacymodel.Hash,
) yacymodel.Optional[peerjudgements.RecordedJudgement] {
	judgement, held := l.judgements.Get(formAndPeer{form: form, peer: peer})
	if !held {
		return yacymodel.None[peerjudgements.RecordedJudgement]()
	}

	return yacymodel.Some(judgement)
}

func (l *JudgementLedger) HoldJudgement(
	_ context.Context,
	judgement peerjudgements.RecordedJudgement,
) {
	l.judgements.Add(formAndPeer{form: judgement.Form, peer: judgement.Peer}, judgement)
}
