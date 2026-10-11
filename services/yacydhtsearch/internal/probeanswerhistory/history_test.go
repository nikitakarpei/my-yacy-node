package probeanswerhistory_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/probeanswerhistory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	networkName      = "yacydhtsearch"
	answeringAddress = "http://10.0.0.1:8090"
	readingDeadline  = 5 * time.Second
	keptFor          = time.Hour
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

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

type historyOnATestServer struct {
	history probeanswerhistory.History
	server  *miniredis.Miniredis
	clock   *manualClock
}

func historyOver(t *testing.T) historyOnATestServer {
	t.Helper()

	server := miniredis.RunT(t)
	server.SetTime(startOfObservation())
	clock := &manualClock{now: startOfObservation()}

	return historyOnATestServer{
		history: probeanswerhistory.New(
			redis.NewClient(&redis.Options{Addr: server.Addr()}),
			networkName,
			keptFor,
			clock,
			probeanswerhistory.HistoryObservers{silentObserver{}},
		),
		server: server,
		clock:  clock,
	}
}

func (h historyOnATestServer) passTime(elapsed time.Duration) {
	h.clock.now = h.clock.now.Add(elapsed)
	h.server.SetTime(h.clock.now)
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

	history := historyOver(t).history
	history.Append(t.Context(), answerOf(t, 'a', startOfObservation()))
	history.Append(t.Context(), answerOf(t, 'b', startOfObservation().Add(time.Minute)))

	_, answers := readAfter(t, history, probeanswerhistory.ProbeAnswerPosition{}, 2)

	if answers[0].Hash != hashOf(t, 'a') || answers[1].Hash != hashOf(t, 'b') {
		t.Fatalf("read %+v, want the answers in the order they were appended", answers)
	}
	if !answers[0].AnsweredAt.Equal(startOfObservation()) {
		t.Fatalf("read %v, want the time the peer answered at", answers[0].AnsweredAt)
	}
}

func TestAReaderReadsOnFromThePositionItKept(t *testing.T) {
	t.Parallel()

	history := historyOver(t).history
	history.Append(t.Context(), answerOf(t, 'a', startOfObservation()))
	history.Append(t.Context(), answerOf(t, 'b', startOfObservation().Add(time.Minute)))
	positions, _ := readAfter(t, history, probeanswerhistory.ProbeAnswerPosition{}, 2)

	_, readAgain := readAfter(t, history, positions[0], 1)

	if readAgain[0].Hash != hashOf(t, 'b') {
		t.Fatalf("read %+v, want the answer after the position that was kept", readAgain[0])
	}
}

func TestAReaderStartsAtTheEarliestAnswerTheHistoryStillHolds(t *testing.T) {
	t.Parallel()

	shared := historyOver(t)
	shared.history.Append(t.Context(), answerOf(t, 'a', startOfObservation()))
	shared.passTime(keptFor + time.Minute)
	shared.history.Append(t.Context(), answerOf(t, 'b', startOfObservation().Add(keptFor)))

	_, answers := readAfter(t, shared.history, probeanswerhistory.ProbeAnswerPosition{}, 1)

	if answers[0].Hash != hashOf(t, 'b') {
		t.Fatalf("read %+v, want the earliest answer the history still holds", answers[0])
	}
}

func TestAPositionReadsBackFromTheTextItIsKeptAs(t *testing.T) {
	t.Parallel()

	history := historyOver(t).history
	history.Append(t.Context(), answerOf(t, 'a', startOfObservation()))
	positions, _ := readAfter(t, history, probeanswerhistory.ProbeAnswerPosition{}, 1)

	text, err := positions[0].MarshalText()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var kept probeanswerhistory.ProbeAnswerPosition
	if err := kept.UnmarshalText(text); err != nil {
		t.Fatalf("unmarshal %q: %v", text, err)
	}

	if kept != positions[0] {
		t.Fatalf("kept %v, want %v", kept, positions[0])
	}
}

func TestTextThatIsNotAPositionIsRefused(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"7", "x-1", "7-y"} {
		var kept probeanswerhistory.ProbeAnswerPosition
		if err := kept.UnmarshalText([]byte(text)); err == nil {
			t.Errorf("UnmarshalText(%q) accepted text that is not a position", text)
		}
	}
}

func TestTheEarliestPositionIsTheOneEveryReaderHasReached(t *testing.T) {
	t.Parallel()

	earliest := probeanswerhistory.EarliestOf(
		probeanswerhistory.ProbeAnswerPosition{AppendedAtMillis: 7},
		probeanswerhistory.ProbeAnswerPosition{AppendedAtMillis: 3, Sequence: 2},
		probeanswerhistory.ProbeAnswerPosition{AppendedAtMillis: 3, Sequence: 1},
		probeanswerhistory.ProbeAnswerPosition{AppendedAtMillis: 9},
	)

	if earliest != (probeanswerhistory.ProbeAnswerPosition{AppendedAtMillis: 3, Sequence: 1}) {
		t.Fatalf("EarliestOf = %v, want the earliest position given", earliest)
	}
}

func TestNoPositionAtAllIsTheStartOfTheHistory(t *testing.T) {
	t.Parallel()

	if earliest := probeanswerhistory.EarliestOf(); !earliest.IsZero() {
		t.Fatalf("EarliestOf = %v, want the start of the history", earliest)
	}
}
