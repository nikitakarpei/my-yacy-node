// Package applog writes each statement signing failure to the application log.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgKeyUnavailable        = "hardware key unavailable, statement not signed"
	msgSignatureRefused      = "hardware key refused to sign, statement not signed"
	msgKeyEvidenceUnreadable = "key evidence of the hardware key unreadable, statement not signed"
	msgTimeStampUnavailable  = "time-stamp unavailable, statement not signed"
)

type SigningLog struct{}

func (SigningLog) KeyUnavailable(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgKeyUnavailable, slog.Any("error", cause))
}

func (SigningLog) SignatureRefused(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgSignatureRefused, slog.Any("error", cause))
}

func (SigningLog) KeyEvidenceUnreadable(ctx context.Context, cause error) {
	slog.ErrorContext(ctx, msgKeyEvidenceUnreadable, slog.Any("error", cause))
}

func (SigningLog) TimeStampUnavailable(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgTimeStampUnavailable, slog.Any("error", cause))
}
