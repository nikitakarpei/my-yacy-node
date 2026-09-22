package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	peerjudgementledgersjetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgementledgers/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerjudgements"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	bucketName                              = "peer-judgements"
	question        peerjudgements.Question = "a question about the peer"
	anotherQuestion peerjudgements.Question = "another question about the peer"
	valueCeiling                            = 16
)

type recordedFailures struct {
	lookups int
	holds   int
}

func (r *recordedFailures) JudgementLookupFailed(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
	error,
) {
	r.lookups++
}

func (r *recordedFailures) JudgementHoldFailed(
	context.Context,
	yacymodel.Hash,
	peerjudgements.Question,
	error,
) {
	r.holds++
}

func judgementOfThePeer(judgement peerjudgements.Judgement) peerjudgements.RecordedJudgement {
	return peerjudgements.RecordedJudgement{
		Question: question,
		JudgedPeer: peerjudgements.JudgedPeer{
			PeerAtVersion: peerjudgements.PeerAtVersion{
				Peer:    yacymodel.WordHash("one"),
				Version: "yacy_v1.925",
			},
			Judgement: judgement,
		},
		JudgedAt: time.Date(2026, time.September, 22, 12, 0, 0, 0, time.UTC),
	}
}

func ledgerOver(
	t *testing.T,
	config natsjetstream.KeyValueConfig,
	failures *recordedFailures,
) *peerjudgementledgersjetstream.JudgementLedger {
	t.Helper()

	stream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	config.Bucket = bucketName
	bucket, err := stream.CreateOrUpdateKeyValue(t.Context(), config)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	return peerjudgementledgersjetstream.New(
		bucket,
		peerjudgementledgersjetstream.JudgementLedgerObservers{failures},
	)
}

func TestAHeldJudgementIsGivenBackForItsPeerAndQuestion(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	ledger := ledgerOver(t, natsjetstream.KeyValueConfig{}, failures)
	judgement := judgementOfThePeer(peerjudgements.Honored)

	ledger.HoldJudgement(t.Context(), judgement)

	heldJudgement, held := ledger.JudgementOf(t.Context(), judgement.Peer, question).Get()
	if !held || heldJudgement != judgement {
		t.Fatalf("JudgementOf = %+v, %v, want the judgement held", heldJudgement, held)
	}
	if *failures != (recordedFailures{}) {
		t.Fatalf("failures reported %+v, want none", failures)
	}
}

func TestNoJudgementIsGivenBackForAPeerNeverJudged(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	ledger := ledgerOver(t, natsjetstream.KeyValueConfig{}, failures)

	held := ledger.JudgementOf(t.Context(), yacymodel.WordHash("one"), question).Present()
	if held || failures.lookups != 0 {
		t.Fatalf("JudgementOf held %v with %d failures, want a plain miss", held, failures.lookups)
	}
}

func TestAJudgementOnOneQuestionDoesNotAnswerAnother(t *testing.T) {
	t.Parallel()

	ledger := ledgerOver(t, natsjetstream.KeyValueConfig{}, &recordedFailures{})
	judgement := judgementOfThePeer(peerjudgements.Honored)

	ledger.HoldJudgement(t.Context(), judgement)

	if ledger.JudgementOf(t.Context(), judgement.Peer, anotherQuestion).Present() {
		t.Fatalf("JudgementOf gave back a judgement on %q, want none", anotherQuestion)
	}
}

func TestTheLatestJudgementOfAPeerReplacesTheEarlierOne(t *testing.T) {
	t.Parallel()

	ledger := ledgerOver(t, natsjetstream.KeyValueConfig{History: 1}, &recordedFailures{})
	latestJudgement := judgementOfThePeer(peerjudgements.Ignored)

	ledger.HoldJudgement(t.Context(), judgementOfThePeer(peerjudgements.Honored))
	ledger.HoldJudgement(t.Context(), latestJudgement)

	heldJudgement, _ := ledger.JudgementOf(t.Context(), latestJudgement.Peer, question).Get()
	if heldJudgement != latestJudgement {
		t.Fatalf("JudgementOf = %+v, want the latest judgement %+v", heldJudgement, latestJudgement)
	}
}

func TestAJudgementTheBucketRefusesIsReported(t *testing.T) {
	t.Parallel()

	failures := &recordedFailures{}
	ledger := ledgerOver(t, natsjetstream.KeyValueConfig{MaxValueSize: valueCeiling}, failures)
	judgement := judgementOfThePeer(peerjudgements.Honored)

	ledger.HoldJudgement(t.Context(), judgement)

	if failures.holds != 1 {
		t.Fatalf("hold failures = %d, want one", failures.holds)
	}
	if ledger.JudgementOf(t.Context(), judgement.Peer, question).Present() {
		t.Fatal("JudgementOf gave back a judgement the bucket refused")
	}
}
