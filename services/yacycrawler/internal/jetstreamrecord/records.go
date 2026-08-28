package jetstreamrecord

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	contendedAttempts = 5
	contendedPause    = 20 * time.Millisecond
	absentRevision    = 0
)

var ErrContended = errors.New("another writer changed the record first")

type Records[R any] struct {
	bucket jetstream.KeyValue
}

func New[R any](bucket jetstream.KeyValue) *Records[R] {
	return &Records[R]{bucket: bucket}
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
	standing, revision, err := r.standingRecordAt(ctx, key)
	if err != nil {
		return absent, false, err
	}
	revised, changed := revise(standing)
	if !changed {
		return standing, false, nil
	}
	if err := r.writeRecord(ctx, key, revised, revision); err != nil {
		return absent, false, err
	}
	return revised, true, nil
}

func (r *Records[R]) RecordAt(ctx context.Context, key string) (R, error) {
	record, _, err := r.standingRecordAt(ctx, key)
	return record, err
}

func (r *Records[R]) standingRecordAt(ctx context.Context, key string) (R, uint64, error) {
	var absent R
	entry, err := r.bucket.Get(ctx, key)
	if errors.Is(err, jetstream.ErrKeyNotFound) {
		return absent, absentRevision, nil
	}
	if err != nil {
		return absent, absentRevision, fmt.Errorf("read the record at %s: %w", key, err)
	}
	var record R
	if err := json.Unmarshal(entry.Value(), &record); err != nil {
		return absent, absentRevision, fmt.Errorf("unmarshal the record at %s: %w", key, err)
	}
	return record, entry.Revision(), nil
}

func (r *Records[R]) writeRecord(
	ctx context.Context,
	key string,
	record R,
	revision uint64,
) error {
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal the record at %s: %w", key, err)
	}
	if revision == absentRevision {
		return r.openRecord(ctx, key, data)
	}
	return r.replaceRecord(ctx, key, data, revision)
}

func (r *Records[R]) openRecord(ctx context.Context, key string, data []byte) error {
	if _, err := r.bucket.Create(ctx, key, data); err != nil {
		return writeFailureAt(key, err)
	}
	return nil
}

func (r *Records[R]) replaceRecord(
	ctx context.Context,
	key string,
	data []byte,
	revision uint64,
) error {
	if _, err := r.bucket.Update(ctx, key, data, revision); err != nil {
		return writeFailureAt(key, err)
	}
	return nil
}

func writeFailureAt(key string, err error) error {
	if errors.Is(err, jetstream.ErrKeyExists) {
		return ErrContended
	}
	return fmt.Errorf("write the record at %s: %w", key, err)
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
