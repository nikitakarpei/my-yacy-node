// Package jetstream holds the judgements of the peers in a NATS key-value
// bucket, one entry per peer and question, so that every service instance
// reads the same judgements and an instance that restarts keeps them.
package jetstream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type JudgementLedger struct {
	bucket   natsjetstream.KeyValue
	observer JudgementLedgerObserver
}

func New(bucket natsjetstream.KeyValue, observer JudgementLedgerObserver) *JudgementLedger {
	return &JudgementLedger{bucket: bucket, observer: observer}
}

func (l *JudgementLedger) JudgementOf(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
) yacymodel.Optional[peerjudgements.RecordedJudgement] {
	entry, err := l.bucket.Get(ctx, keyOf(peer, question))
	if errors.Is(err, natsjetstream.ErrKeyNotFound) {
		return yacymodel.None[peerjudgements.RecordedJudgement]()
	}
	if err != nil {
		l.observer.JudgementLookupFailed(ctx, peer, question, err)

		return yacymodel.None[peerjudgements.RecordedJudgement]()
	}

	var judgement peerjudgements.RecordedJudgement
	if err := json.Unmarshal(entry.Value(), &judgement); err != nil {
		l.observer.JudgementLookupFailed(ctx, peer, question, err)

		return yacymodel.None[peerjudgements.RecordedJudgement]()
	}

	return yacymodel.Some(judgement)
}

func (l *JudgementLedger) HoldJudgement(
	ctx context.Context,
	judgement peerjudgements.RecordedJudgement,
) {
	encoded, err := json.Marshal(judgement)
	if err != nil {
		l.observer.JudgementHoldFailed(ctx, judgement.Peer, judgement.Question, err)

		return
	}
	if _, err := l.bucket.Put(ctx, keyOf(judgement.Peer, judgement.Question), encoded); err != nil {
		l.observer.JudgementHoldFailed(ctx, judgement.Peer, judgement.Question, err)
	}
}

func keyOf(peer yacymodel.Hash, question peerjudgements.Question) string {
	return peer.String() + "." + base64.RawURLEncoding.EncodeToString([]byte(question))
}
