package jetstream_test

import (
	"context"
	"sync"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	peerjudgementledgersjetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/jetstream"
	peerjudgementledgersmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	bucketName                                          = "peer-judgements"
	question                    peerjudgements.Question = "a question about the peer"
	anotherQuestion             peerjudgements.Question = "another question about the peer"
	mirrorCapacity                                      = 16
	valueCeiling                                        = 16
	amountOfHoldsBeyondAnyQueue                         = 1 << 16
	waitLimit                                           = 10 * time.Second
)

type recordedFailures struct {
	mutex sync.Mutex
	drops int
	holds int
	other int
}

func (r *recordedFailures) JudgementDropped(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.drops++
}

func (r *recordedFailures) JudgementHoldFailed(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
	error,
) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.holds++
}

func (r *recordedFailures) WatchFailed(context.Context, error) {
	r.countOther()
}

func (r *recordedFailures) JudgementUndecodable(context.Context, string, error) {
	r.countOther()
}

func (r *recordedFailures) WatchEnded(context.Context) {
	r.countOther()
}

func (r *recordedFailures) countOther() {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.other++
}

func (r *recordedFailures) amountOfDrops() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.drops
}

func (r *recordedFailures) amountOfHolds() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.holds
}

func (r *recordedFailures) amountOfFailures() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.drops + r.holds + r.other
}

func judgementOf(peer string, judgement peerjudgements.Judgement) peerjudgements.RecordedJudgement {
	return peerjudgements.RecordedJudgement{
		Question: question,
		JudgedPeer: peerjudgements.JudgedPeer{
			PeerAtVersion: peerjudgements.PeerAtVersion{
				Peer:    yacymodel.WordHash(peer),
				Version: yacymodel.Some(yacymodel.SoftwareVersion{Release: 1.925}),
			},
			Judgement: judgement,
		},
		JudgedAt: time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC),
	}
}

func bucketWith(t *testing.T, config natsjetstream.KeyValueConfig) natsjetstream.KeyValue {
	t.Helper()

	stream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	config.Bucket = bucketName
	bucket, err := stream.CreateOrUpdateKeyValue(t.Context(), config)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	return bucket
}

func ledgerOver(
	bucket natsjetstream.KeyValue,
	failures *recordedFailures,
) *peerjudgementledgersjetstream.JudgementLedger {
	return peerjudgementledgersjetstream.New(
		bucket,
		peerjudgementledgersmemory.New(mirrorCapacity),
		peerjudgementledgersjetstream.JudgementLedgerObservers{failures},
	)
}

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(waitLimit)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("the condition did not hold before the deadline")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestAHeldJudgementIsGivenBackAtOnce(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	ledger := ledgerOver(bucketWith(t, natsjetstream.KeyValueConfig{}), failures)
	judgement := judgementOf("one", peerjudgements.Honored)

	ledger.HoldJudgement(t.Context(), judgement)

	heldJudgement, held := ledger.JudgementOf(t.Context(), judgement.Peer, question).Get()
	if !held || heldJudgement != judgement {
		t.Fatalf("JudgementOf = %+v, %v, want the judgement held", heldJudgement, held)
	}
	if failures.amountOfFailures() != 0 {
		t.Fatalf("failures reported %d, want none", failures.amountOfFailures())
	}
}

func TestNoJudgementIsGivenBackForAPeerNeverJudged(t *testing.T) {
	t.Parallel()

	ledger := ledgerOver(bucketWith(t, natsjetstream.KeyValueConfig{}), &recordedFailures{})

	if ledger.JudgementOf(t.Context(), yacymodel.WordHash("one"), question).Present() {
		t.Fatal("JudgementOf gave back a judgement of a peer never judged")
	}
}

func TestAJudgementOnOneQuestionDoesNotAnswerAnother(t *testing.T) {
	t.Parallel()

	ledger := ledgerOver(bucketWith(t, natsjetstream.KeyValueConfig{}), &recordedFailures{})
	judgement := judgementOf("one", peerjudgements.Honored)

	ledger.HoldJudgement(t.Context(), judgement)

	if ledger.JudgementOf(t.Context(), judgement.Peer, anotherQuestion).Present() {
		t.Fatalf("JudgementOf gave back a judgement on %q, want none", anotherQuestion)
	}
}

func TestAJudgementHeldByAnotherInstanceIsGivenBackOnceWatched(t *testing.T) {
	t.Parallel()

	bucket := bucketWith(t, natsjetstream.KeyValueConfig{})
	judgingLedger := ledgerOver(bucket, &recordedFailures{})
	readingLedger := ledgerOver(bucket, &recordedFailures{})
	go judgingLedger.ShareTheJudgements(t.Context())
	go readingLedger.ShareTheJudgements(t.Context())
	judgement := judgementOf("one", peerjudgements.Honored)

	judgingLedger.HoldJudgement(t.Context(), judgement)

	waitUntil(t, func() bool {
		heldJudgement, held := readingLedger.JudgementOf(t.Context(), judgement.Peer, question).
			Get()

		return held && heldJudgement == judgement
	})
}

func TestAJudgementHeldBeforeTheSharingStartsReachesTheBucket(t *testing.T) {
	t.Parallel()

	bucket := bucketWith(t, natsjetstream.KeyValueConfig{})
	ledger := ledgerOver(bucket, &recordedFailures{})
	judgement := judgementOf("one", peerjudgements.Honored)

	ledger.HoldJudgement(t.Context(), judgement)
	go ledger.ShareTheJudgements(t.Context())

	waitUntil(t, func() bool {
		keys, err := bucket.ListKeys(t.Context())
		if err != nil {
			return false
		}
		amountOfKeys := 0
		for range keys.Keys() {
			amountOfKeys++
		}

		return amountOfKeys == 1
	})
}

func TestAnEarlierJudgementFromTheBucketDoesNotReplaceALaterOne(t *testing.T) {
	t.Parallel()

	bucket := bucketWith(t, natsjetstream.KeyValueConfig{History: 1})
	readingLedger := ledgerOver(bucket, &recordedFailures{})
	judgingLedger := ledgerOver(bucket, &recordedFailures{})
	go readingLedger.ShareTheJudgements(t.Context())
	go judgingLedger.ShareTheJudgements(t.Context())
	earlierJudgement := judgementOf("one", peerjudgements.Honored)
	laterJudgement := judgementOf("one", peerjudgements.Ignored)
	laterJudgement.JudgedAt = earlierJudgement.JudgedAt.Add(time.Hour)
	judgementWatchedLast := judgementOf("two", peerjudgements.Honored)

	readingLedger.HoldJudgement(t.Context(), laterJudgement)
	judgingLedger.HoldJudgement(t.Context(), earlierJudgement)
	judgingLedger.HoldJudgement(t.Context(), judgementWatchedLast)
	waitUntil(t, func() bool {
		return readingLedger.JudgementOf(t.Context(), judgementWatchedLast.Peer, question).Present()
	})

	heldJudgement, _ := readingLedger.JudgementOf(t.Context(), laterJudgement.Peer, question).Get()
	if heldJudgement != laterJudgement {
		t.Fatalf("JudgementOf = %+v, want the later judgement %+v", heldJudgement, laterJudgement)
	}
}

func TestAJudgementTheBucketRefusesIsReported(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	ledger := ledgerOver(
		bucketWith(t, natsjetstream.KeyValueConfig{MaxValueSize: valueCeiling}),
		failures,
	)
	go ledger.ShareTheJudgements(t.Context())

	ledger.HoldJudgement(t.Context(), judgementOf("one", peerjudgements.Honored))

	waitUntil(t, func() bool { return failures.amountOfHolds() == 1 })
}

func TestAJudgementBeyondAFullQueueIsReportedDropped(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	ledger := ledgerOver(bucketWith(t, natsjetstream.KeyValueConfig{}), failures)
	judgement := judgementOf("one", peerjudgements.Honored)

	for range amountOfHoldsBeyondAnyQueue {
		ledger.HoldJudgement(t.Context(), judgement)
	}

	if failures.amountOfDrops() == 0 {
		t.Fatal("no judgement was reported dropped, want the ones beyond the queue")
	}
}
