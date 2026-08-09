package vault

import "time"

type TransactionObserver interface {
	ObserveWriteBegan(elapsed time.Duration)
	ObserveWriteBeginRefused(cause WriteRefusalCause)
	ObserveWriteCommitted(executed, committed time.Duration)
	ObserveWriteAborted(executed, rolledBack time.Duration)
	ObserveWriteCommitRefused(executed, rolledBack time.Duration, cause WriteRefusalCause)
	ObserveReadBegan()
	ObserveReadEnded()
}

type silentObserver struct{}

func (silentObserver) ObserveWriteBegan(time.Duration) {}

func (silentObserver) ObserveWriteBeginRefused(WriteRefusalCause) {}

func (silentObserver) ObserveWriteCommitted(time.Duration, time.Duration) {}

func (silentObserver) ObserveWriteAborted(time.Duration, time.Duration) {}

func (silentObserver) ObserveWriteCommitRefused(time.Duration, time.Duration, WriteRefusalCause) {}

func (silentObserver) ObserveReadBegan() {}

func (silentObserver) ObserveReadEnded() {}
