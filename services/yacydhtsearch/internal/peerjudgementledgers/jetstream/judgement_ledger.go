// Package jetstream holds the judgements of the peers in a NATS key-value
// bucket, one entry per peer and question, so that every service instance
// reads the same judgements and an instance that restarts keeps them. Each
// instance reads the judgements from a copy in memory that follows the bucket,
// so the first queries after a start can run before the copy holds any of them.
package jetstream

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	peerjudgementledgersmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	queueLength         = 1024
	keyOfEveryJudgement = "*.*"
	pauseBeforeWatch    = 5 * time.Second
)

type JudgementLedger struct {
	bucket           natsjetstream.KeyValue
	mirror           *peerjudgementledgersmemory.JudgementLedger
	queuedJudgements chan peerjudgements.RecordedJudgement
	observer         JudgementLedgerObserver
}

func New(
	bucket natsjetstream.KeyValue,
	mirror *peerjudgementledgersmemory.JudgementLedger,
	observer JudgementLedgerObserver,
) *JudgementLedger {
	return &JudgementLedger{
		bucket:           bucket,
		mirror:           mirror,
		queuedJudgements: make(chan peerjudgements.RecordedJudgement, queueLength),
		observer:         observer,
	}
}

func (l *JudgementLedger) JudgementOf(
	ctx context.Context,
	peer yacymodel.Hash,
	question peerjudgements.Question,
) yacymodel.Optional[peerjudgements.RecordedJudgement] {
	return l.mirror.JudgementOf(ctx, peer, question)
}

func (l *JudgementLedger) HoldJudgement(
	ctx context.Context,
	judgement peerjudgements.RecordedJudgement,
) {
	l.mirror.HoldJudgement(ctx, judgement)
	l.queue(ctx, judgement)
}

func (l *JudgementLedger) queue(ctx context.Context, judgement peerjudgements.RecordedJudgement) {
	select {
	case l.queuedJudgements <- judgement:
	default:
		l.observer.JudgementDropped(ctx, judgement.Peer, judgement.Question)
	}
}

func (l *JudgementLedger) ShareTheJudgements(ctx context.Context) {
	go l.putQueuedJudgements(ctx)
	l.followTheBucket(ctx)
}

func (l *JudgementLedger) putQueuedJudgements(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case judgement := <-l.queuedJudgements:
			l.put(ctx, judgement)
		}
	}
}

func (l *JudgementLedger) put(ctx context.Context, judgement peerjudgements.RecordedJudgement) {
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

func (l *JudgementLedger) followTheBucket(ctx context.Context) {
	for ctx.Err() == nil {
		l.foldWatchedEntries(ctx)
		waitForTheNextWatch(ctx)
	}
}

func (l *JudgementLedger) foldWatchedEntries(ctx context.Context) {
	watcher, err := l.bucket.Watch(ctx, keyOfEveryJudgement, natsjetstream.IgnoreDeletes())
	if err != nil {
		l.observer.WatchFailed(ctx, err)

		return
	}
	defer func() { _ = watcher.Stop() }()

	for entry := range watcher.Updates() {
		if entry != nil {
			l.fold(ctx, entry)
		}
	}
	if ctx.Err() == nil {
		l.observer.WatchEnded(ctx)
	}
}

func (l *JudgementLedger) fold(ctx context.Context, entry natsjetstream.KeyValueEntry) {
	judgement, decoded := l.judgementIn(ctx, entry).Get()
	if decoded && !l.isOutdated(ctx, judgement) {
		l.mirror.HoldJudgement(ctx, judgement)
	}
}

func (l *JudgementLedger) judgementIn(
	ctx context.Context,
	entry natsjetstream.KeyValueEntry,
) yacymodel.Optional[peerjudgements.RecordedJudgement] {
	var judgement peerjudgements.RecordedJudgement
	if err := json.Unmarshal(entry.Value(), &judgement); err != nil {
		l.observer.JudgementUndecodable(ctx, entry.Key(), err)

		return yacymodel.None[peerjudgements.RecordedJudgement]()
	}

	return yacymodel.Some(judgement)
}

func (l *JudgementLedger) isOutdated(
	ctx context.Context,
	judgement peerjudgements.RecordedJudgement,
) bool {
	heldJudgement, held := l.mirror.JudgementOf(ctx, judgement.Peer, judgement.Question).Get()

	return held && heldJudgement.JudgedAt.After(judgement.JudgedAt)
}

func waitForTheNextWatch(ctx context.Context) {
	pause := time.NewTimer(pauseBeforeWatch)
	defer pause.Stop()

	select {
	case <-ctx.Done():
	case <-pause.C:
	}
}
