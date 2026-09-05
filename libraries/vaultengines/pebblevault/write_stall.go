package pebblevault

import (
	"context"
	"log/slog"
)

type WriteStallCause string

const (
	causeUnclassified           WriteStallCause = "unclassified"
	causeMemtablesAwaitingFlush WriteStallCause = "memtables_awaiting_flush"
)

const causeLevelZeroFilesAwaitingCompaction WriteStallCause = "level_zero_files_awaiting_compaction"

const (
	reasonMemtablesAwaitingFlush           = "memtable count limit reached"
	reasonLevelZeroFilesAwaitingCompaction = "L0 file count limit exceeded"
)

type WriteStallObserver interface {
	ObserveWriteStallBegan(cause WriteStallCause)
	ObserveWriteStallEnded()
}

type silentWriteStallObserver struct{}

func (silentWriteStallObserver) ObserveWriteStallBegan(WriteStallCause) {}

func (silentWriteStallObserver) ObserveWriteStallEnded() {}

func writeStallCauseOf(reason string) WriteStallCause {
	switch reason {
	case reasonMemtablesAwaitingFlush:
		return causeMemtablesAwaitingFlush
	case reasonLevelZeroFilesAwaitingCompaction:
		return causeLevelZeroFilesAwaitingCompaction
	}

	return causeUnclassified
}

func reportWriteStallBegan(observer WriteStallObserver, cause WriteStallCause) {
	observer.ObserveWriteStallBegan(cause)
	slog.WarnContext(
		context.Background(),
		"storage delayed writes",
		slog.String("cause", string(cause)),
	)
}

func reportWriteStallEnded(observer WriteStallObserver) {
	observer.ObserveWriteStallEnded()
	slog.DebugContext(context.Background(), "storage released delayed writes")
}
