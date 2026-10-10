package pebblevault

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type WriteConflictObserver interface {
	ObserveWriteConflicted(bucket vault.Name, kind KeyKind)
	ObserveExclusiveWrite()
}

type silentWriteConflictObserver struct{}

func (silentWriteConflictObserver) ObserveWriteConflicted(vault.Name, KeyKind) {}

func (silentWriteConflictObserver) ObserveExclusiveWrite() {}

func writeConflictObserverOrSilent(observer WriteConflictObserver) WriteConflictObserver {
	if observer == nil {
		return silentWriteConflictObserver{}
	}

	return observer
}

func reportWriteConflicted(ctx context.Context, observer WriteConflictObserver, key []byte) {
	bucket, kind := bucketAndKindOf(key)
	observer.ObserveWriteConflicted(bucket, kind)
	slog.DebugContext(
		ctx,
		"write read a key another write committed after it began",
		slog.String("bucket", string(bucket)),
		slog.String("kind", string(kind)),
	)
}

func reportExclusiveWrite(ctx context.Context, observer WriteConflictObserver) {
	observer.ObserveExclusiveWrite()
	slog.DebugContext(ctx, "write holds other commits after it conflicted too often")
}
