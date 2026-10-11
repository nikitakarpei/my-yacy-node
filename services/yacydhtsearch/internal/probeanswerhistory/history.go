// Package probeanswerhistory keeps every answer a peer gave to a probe, in the
// order this deployment observed them, in a stream on a Redis-protocol server that
// all instances of the service share. An instance appends the answers it observes
// and reads every answer back, its own among them, so all instances read the same
// answers in the same order. Each answer comes back with the position it holds
// in the history, and a consumer that keeps a position reads on from there
// when it starts again. The history drops answers older than the time it keeps
// them for, so the earliest answer it still holds can be later than the
// position a consumer asks from.
package probeanswerhistory

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix      = "yacydhtsearch:probe-answers:"
	answerField    = "answer"
	answerWait     = time.Second
	answersPerRead = 100
)

type Clock interface {
	Now() time.Time
}

type History struct {
	client      redis.UniversalClient
	networkName string
	keptFor     time.Duration
	clock       Clock
	observer    HistoryObserver
}

func New(
	client redis.UniversalClient,
	networkName string,
	keptFor time.Duration,
	clock Clock,
	observer HistoryObserver,
) History {
	return History{
		client:      client,
		networkName: networkName,
		keptFor:     keptFor,
		clock:       clock,
		observer:    observer,
	}
}

func (h History) Append(ctx context.Context, answer ProbeAnswer) {
	encoded, err := json.Marshal(answer)
	if err != nil {
		h.observer.ProbeAnswerAppendFailed(ctx, err)

		return
	}
	if err := h.client.XAdd(ctx, &redis.XAddArgs{
		Stream: h.key(),
		MinID:  strconv.FormatInt(h.clock.Now().Add(-h.keptFor).UnixMilli(), 10),
		Values: []any{answerField, encoded},
	}).Err(); err != nil {
		h.observer.ProbeAnswerAppendFailed(ctx, err)
	}
}

func (h History) AnswersAfter(
	ctx context.Context,
	position ProbeAnswerPosition,
) iter.Seq2[ProbeAnswerPosition, ProbeAnswer] {
	return func(yield func(ProbeAnswerPosition, ProbeAnswer) bool) {
		for ctx.Err() == nil {
			entries, err := h.entriesAfter(ctx, position)
			if ctx.Err() != nil {
				return
			}
			if err != nil {
				h.observer.ProbeAnswerHistoryEnded(ctx, err)

				return
			}
			for _, entry := range entries {
				answered, answer, readable := h.answerIn(ctx, entry)
				if !answered.IsZero() {
					position = answered
				}
				if readable && !yield(answered, answer) {
					return
				}
			}
		}
	}
}

func (h History) entriesAfter(
	ctx context.Context,
	position ProbeAnswerPosition,
) ([]redis.XMessage, error) {
	streams, err := h.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{h.key(), position.String()},
		Count:   answersPerRead,
		Block:   answerWait,
	}).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err //nolint:wrapcheck // the caller reports it to the history observer
	}
	if len(streams) == 0 {
		return nil, nil
	}

	return streams[0].Messages, nil
}

func (h History) answerIn(
	ctx context.Context,
	entry redis.XMessage,
) (ProbeAnswerPosition, ProbeAnswer, bool) {
	position, err := ProbeAnswerPositionOf(entry.ID)
	if err != nil {
		h.observer.ProbeAnswerUnreadable(ctx, ProbeAnswerPosition{}, err)

		return ProbeAnswerPosition{}, ProbeAnswer{}, false
	}
	encoded, held := entry.Values[answerField].(string)
	if !held {
		h.observer.ProbeAnswerUnreadable(ctx, position, errAnswerMissing)

		return position, ProbeAnswer{}, false
	}
	var answer ProbeAnswer
	if err := json.Unmarshal([]byte(encoded), &answer); err != nil {
		h.observer.ProbeAnswerUnreadable(ctx, position, err)

		return position, ProbeAnswer{}, false
	}

	return position, answer, true
}

var errAnswerMissing = errors.New("the history entry holds no answer")

func (h History) key() string {
	return keyPrefix + h.networkName
}
