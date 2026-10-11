package redisrecord

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	contendedAttempts = 5
	contendedPause    = 20 * time.Millisecond
	existingFlag      = "1"
)

var ErrContended = errors.New("another writer changed the record first")

var replaceStandingRecord = redis.NewScript(`
local standing = redis.call('GET', KEYS[1])
if ARGV[1] == '' then
  if standing then return 0 end
elseif standing ~= ARGV[2] then
  return 0
end
redis.call('SET', KEYS[1], ARGV[3], 'PX', ARGV[4])
return 1
`)

type Records[R any] struct {
	client    redis.UniversalClient
	retention time.Duration
}

func New[R any](client redis.UniversalClient, retention time.Duration) *Records[R] {
	return &Records[R]{client: client, retention: retention}
}

func (r *Records[R]) Revise(
	ctx context.Context,
	key string,
	revise func(R) (R, bool),
) (R, bool, error) {
	var absent R
	for attempt := range contendedAttempts {
		standing, revised, err := r.reviseOnce(ctx, key, revise)
		if errors.Is(err, ErrContended) {
			if err := pauseAfterContention(ctx, attempt); err != nil {
				return absent, false, err
			}
			continue
		}
		if err != nil {
			return absent, false, err
		}
		return standing, revised, nil
	}
	return absent, false, fmt.Errorf("revise the record at %s: %w", key, ErrContended)
}

func (r *Records[R]) reviseOnce(
	ctx context.Context,
	key string,
	revise func(R) (R, bool),
) (R, bool, error) {
	var absent R
	standing, standingRevision, err := r.standingRecordAt(ctx, key)
	if err != nil {
		return absent, false, err
	}
	revised, changed := revise(standing)
	if !changed {
		return standing, false, nil
	}
	if err := r.writeRecord(ctx, key, revised, standingRevision); err != nil {
		return absent, false, err
	}
	return revised, true, nil
}

func (r *Records[R]) RecordAt(ctx context.Context, key string) (R, error) {
	record, _, err := r.standingRecordAt(ctx, key)
	return record, err
}

type revision struct {
	payload string
	exists  bool
}

func (r *Records[R]) standingRecordAt(ctx context.Context, key string) (R, revision, error) {
	var absent R
	payload, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return absent, revision{}, nil
	}
	if err != nil {
		return absent, revision{}, fmt.Errorf("read the record at %s: %w", key, err)
	}
	var record R
	if err := json.Unmarshal([]byte(payload), &record); err != nil {
		return absent, revision{}, fmt.Errorf("unmarshal the record at %s: %w", key, err)
	}
	return record, revision{payload: payload, exists: true}, nil
}

func (r *Records[R]) writeRecord(
	ctx context.Context,
	key string,
	record R,
	standing revision,
) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal the record at %s: %w", key, err)
	}
	replaced, err := replaceStandingRecord.Run(ctx, r.client, []string{key},
		existenceFlagOf(standing), standing.payload, data, r.retention.Milliseconds(),
	).Int()
	if err != nil {
		return fmt.Errorf("write the record at %s: %w", key, err)
	}
	if replaced == 0 {
		return ErrContended
	}
	return nil
}

func existenceFlagOf(standing revision) string {
	if standing.exists {
		return existingFlag
	}
	return ""
}

func pauseAfterContention(ctx context.Context, attempt int) error {
	pause := contendedPause << attempt
	timer := time.NewTimer(pause/2 + jitterWithin(pause/2))
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("wait out the writer that won the record: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func jitterWithin(span time.Duration) time.Duration {
	drawn, err := rand.Int(rand.Reader, big.NewInt(int64(span)))
	if err != nil {
		return span
	}
	return time.Duration(drawn.Int64())
}
