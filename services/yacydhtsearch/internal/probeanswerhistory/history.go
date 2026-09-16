// Package probeanswerhistory keeps every answer a peer gave to a probe, in the
// order this deployment observed them, in a NATS stream that all instances of
// the service share. An instance appends the answers it observes and reads
// every answer back, its own among them, so all instances read the same
// answers in the same order. Each answer comes back with the position it holds
// in the history, and a consumer that keeps a position reads on from there
// when it starts again. The history holds the answers of a recent while and
// not of all time, so the earliest answer it still holds can be later than the
// position a consumer asks from.
package probeanswerhistory

import (
	"context"
	"encoding/json"
	"iter"

	natsjetstream "github.com/nats-io/nats.go/jetstream"
)

type History struct {
	jetStream   natsjetstream.JetStream
	streamName  string
	networkName string
	observer    HistoryObserver
}

func New(
	jetStream natsjetstream.JetStream,
	streamName string,
	networkName string,
	observer HistoryObserver,
) History {
	return History{
		jetStream:   jetStream,
		streamName:  streamName,
		networkName: networkName,
		observer:    observer,
	}
}

func (h History) Append(ctx context.Context, answer ProbeAnswer) {
	encoded, err := json.Marshal(answer)
	if err != nil {
		h.observer.ProbeAnswerAppendFailed(ctx, err)

		return
	}
	subject := subjectOfProbeAnswerIn(h.networkName, answer.PeerAtAddress)
	if _, err := h.jetStream.Publish(ctx, subject, encoded); err != nil {
		h.observer.ProbeAnswerAppendFailed(ctx, err)
	}
}

func (h History) AnswersAfter(
	ctx context.Context,
	position ProbeAnswerPosition,
) iter.Seq2[ProbeAnswerPosition, ProbeAnswer] {
	return func(yield func(ProbeAnswerPosition, ProbeAnswer) bool) {
		answers, err := h.answersFrom(ctx, h.firstPositionHeldAfter(ctx, position))
		if err != nil {
			h.observer.ProbeAnswerHistoryEnded(ctx, err)

			return
		}
		defer context.AfterFunc(ctx, answers.Stop)()
		defer answers.Stop()

		for {
			message, err := answers.Next()
			if err != nil {
				h.observer.ProbeAnswerHistoryEnded(ctx, err)

				return
			}
			answered, answer, readable := h.answerIn(ctx, message)
			if readable && !yield(answered, answer) {
				return
			}
		}
	}
}

func (h History) firstPositionHeldAfter(
	ctx context.Context,
	position ProbeAnswerPosition,
) ProbeAnswerPosition {
	earliestHeld, historyRead := h.earliestPositionHeld(ctx)
	if historyRead && earliestHeld > position.next() {
		return earliestHeld
	}

	return position.next()
}

func (h History) earliestPositionHeld(ctx context.Context) (ProbeAnswerPosition, bool) {
	stream, err := h.jetStream.Stream(ctx, h.streamName)
	if err != nil {
		h.observer.ProbeAnswerHistoryEnded(ctx, err)

		return 0, false
	}
	streamState, err := stream.Info(ctx)
	if err != nil {
		h.observer.ProbeAnswerHistoryEnded(ctx, err)

		return 0, false
	}

	return ProbeAnswerPosition(streamState.State.FirstSeq), true
}

func (h History) answersFrom(
	ctx context.Context,
	position ProbeAnswerPosition,
) (natsjetstream.MessagesContext, error) {
	consumer, err := h.jetStream.OrderedConsumer(
		ctx,
		h.streamName,
		natsjetstream.OrderedConsumerConfig{
			FilterSubjects: []string{SubjectOfEveryProbeAnswerIn(h.networkName)},
			DeliverPolicy:  natsjetstream.DeliverByStartSequencePolicy,
			OptStartSeq:    uint64(position),
		},
	)
	if err != nil {
		return nil, err //nolint:wrapcheck // the caller reports it to the history observer
	}

	return consumer.Messages() //nolint:wrapcheck // the caller reports it to the history observer
}

func (h History) answerIn(
	ctx context.Context,
	message natsjetstream.Msg,
) (ProbeAnswerPosition, ProbeAnswer, bool) {
	delivered, err := message.Metadata()
	if err != nil {
		h.observer.ProbeAnswerUnreadable(ctx, 0, err)

		return 0, ProbeAnswer{}, false
	}
	position := ProbeAnswerPosition(delivered.Sequence.Stream)
	var answer ProbeAnswer
	if err := json.Unmarshal(message.Data(), &answer); err != nil {
		h.observer.ProbeAnswerUnreadable(ctx, position, err)

		return 0, ProbeAnswer{}, false
	}

	return position, answer, true
}
