package probeanswerhistory_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	networkName      = "yacydhtsearch"
	streamName       = "probe-answer-history"
	answeringAddress = "http://10.0.0.1:8090"
	readingDeadline  = 5 * time.Second
)

type silentObserver struct{}

func (silentObserver) ProbeAnswerAppendFailed(context.Context, error) {}
func (silentObserver) ProbeAnswerUnreadable(
	context.Context,
	probeanswerhistory.ProbeAnswerPosition,
	error,
) {
}
func (silentObserver) ProbeAnswerHistoryEnded(context.Context, error) {}

func startOfObservation() time.Time {
	return time.Date(2026, time.September, 15, 12, 0, 0, 0, time.UTC)
}

func hashOf(t *testing.T, symbol byte) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(string([]byte{
		symbol, symbol, symbol, symbol, symbol, symbol,
		symbol, symbol, symbol, symbol, symbol, symbol,
	}))
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return hash
}

func historyOver(t *testing.T) (probeanswerhistory.History, natsjetstream.JetStream) {
	t.Helper()

	stream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	_, err := stream.CreateOrUpdateStream(t.Context(), natsjetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{probeanswerhistory.SubjectOfEveryProbeAnswerIn(networkName)},
	})
	if err != nil {
		t.Fatalf("create stream: %v", err)
	}

	return probeanswerhistory.New(
		stream, streamName, networkName, probeanswerhistory.HistoryObservers{silentObserver{}},
	), stream
}

func answerOf(t *testing.T, symbol byte, at time.Time) probeanswerhistory.ProbeAnswer {
	t.Helper()

	return probeanswerhistory.ProbeAnswer{
		PeerAtAddress: probeanswerhistory.PeerAtAddress{
			Hash:    hashOf(t, symbol),
			Address: answeringAddress,
		},
		AnsweredAt: at,
	}
}

func readAfter(
	t *testing.T,
	history probeanswerhistory.History,
	position probeanswerhistory.ProbeAnswerPosition,
	amount int,
) ([]probeanswerhistory.ProbeAnswerPosition, []probeanswerhistory.ProbeAnswer) {
	t.Helper()

	read, giveUp := context.WithTimeout(t.Context(), readingDeadline)
	defer giveUp()

	var positions []probeanswerhistory.ProbeAnswerPosition
	var answers []probeanswerhistory.ProbeAnswer
	for readPosition, answer := range history.AnswersAfter(read, position) {
		positions = append(positions, readPosition)
		answers = append(answers, answer)
		if len(answers) == amount {
			break
		}
	}
	if len(answers) != amount {
		t.Fatalf("read %d answers after %v, want %d", len(answers), readingDeadline, amount)
	}

	return positions, answers
}

func TestEveryAppendedAnswerIsReadBackInTheOrderItWasAppended(t *testing.T) {
	t.Parallel()

	history, _ := historyOver(t)
	history.Append(t.Context(), answerOf(t, 'a', startOfObservation()))
	history.Append(t.Context(), answerOf(t, 'b', startOfObservation().Add(time.Minute)))

	_, answers := readAfter(t, history, 0, 2)

	if answers[0].Hash != hashOf(t, 'a') || answers[1].Hash != hashOf(t, 'b') {
		t.Fatalf("read %+v, want the answers in the order they were appended", answers)
	}
	if !answers[0].AnsweredAt.Equal(startOfObservation()) {
		t.Fatalf("read %v, want the time the peer answered at", answers[0].AnsweredAt)
	}
}

func TestAReaderReadsOnFromThePositionItKept(t *testing.T) {
	t.Parallel()

	history, _ := historyOver(t)
	history.Append(t.Context(), answerOf(t, 'a', startOfObservation()))
	history.Append(t.Context(), answerOf(t, 'b', startOfObservation().Add(time.Minute)))
	positions, _ := readAfter(t, history, 0, 2)

	_, readAgain := readAfter(t, history, positions[0], 1)

	if readAgain[0].Hash != hashOf(t, 'b') {
		t.Fatalf("read %+v, want the answer after the position that was kept", readAgain[0])
	}
}

func TestAReaderStartsAtTheEarliestAnswerTheHistoryStillHolds(t *testing.T) {
	t.Parallel()

	history, stream := historyOver(t)
	history.Append(t.Context(), answerOf(t, 'a', startOfObservation()))
	readAfter(t, history, 0, 1)
	dropEveryAnswer(t, stream)
	history.Append(t.Context(), answerOf(t, 'b', startOfObservation().Add(time.Minute)))

	_, answers := readAfter(t, history, 0, 1)

	if answers[0].Hash != hashOf(t, 'b') {
		t.Fatalf("read %+v, want the earliest answer the history still holds", answers[0])
	}
}

func dropEveryAnswer(t *testing.T, stream natsjetstream.JetStream) {
	t.Helper()

	answers, err := stream.Stream(t.Context(), streamName)
	if err != nil {
		t.Fatalf("open stream: %v", err)
	}
	if err := answers.Purge(t.Context()); err != nil {
		t.Fatalf("drop the answers: %v", err)
	}
}

func TestTheEarliestPositionIsTheOneEveryReaderHasReached(t *testing.T) {
	t.Parallel()

	earliest := probeanswerhistory.EarliestOf(7, 3, 9)

	if earliest != 3 {
		t.Fatalf("EarliestOf = %v, want the earliest position given", earliest)
	}
}

func TestNoPositionAtAllIsTheStartOfTheHistory(t *testing.T) {
	t.Parallel()

	if earliest := probeanswerhistory.EarliestOf(); earliest != 0 {
		t.Fatalf("EarliestOf = %v, want the start of the history", earliest)
	}
}
