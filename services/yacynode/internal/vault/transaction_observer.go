package vault

import (
	"errors"
	"time"
)

const unclassifiedCause = "unclassified"

type WriteRefusalCause string

// WriteRefusal is an error the storage engine attaches a Cause to. The engine
// implementing it declares a bounded set of causes, since the value becomes a
// metric label.
type WriteRefusal interface {
	error
	Cause() WriteRefusalCause
}

type TransactionObserver interface {
	ObserveWriteBegan(elapsed time.Duration)
	ObserveWriteBeginRefused(cause WriteRefusalCause)
	ObserveWriteCommitted(executed, committed time.Duration)
	ObserveWriteAborted(executed, rolledBack time.Duration)
	ObserveWriteCommitRefused(executed, rolledBack time.Duration, cause WriteRefusalCause)
	ObserveReadBegan()
	ObserveReadEnded()
}

func refusalCauseOf(err error) WriteRefusalCause {
	var refusal WriteRefusal
	if errors.As(err, &refusal) {
		return refusal.Cause()
	}

	return unclassifiedCause
}

type silentObserver struct{}

func (silentObserver) ObserveWriteBegan(time.Duration) {}

func (silentObserver) ObserveWriteBeginRefused(WriteRefusalCause) {}

func (silentObserver) ObserveWriteCommitted(time.Duration, time.Duration) {}

func (silentObserver) ObserveWriteAborted(time.Duration, time.Duration) {}

func (silentObserver) ObserveWriteCommitRefused(time.Duration, time.Duration, WriteRefusalCause) {}

func (silentObserver) ObserveReadBegan() {}

func (silentObserver) ObserveReadEnded() {}
