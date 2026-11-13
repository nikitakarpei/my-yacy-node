// Package applog writes each refresh of the trusted app releases, and each failure of one, to
// the application log.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgRefreshed        = "trusted app releases refreshed"
	msgDenialsUnread    = "denied app releases unread, trusted app releases not refreshed"
	msgReleasesUnlisted = "app releases not listed, trusted app releases not refreshed"
	msgReleaseUnread    = "app release unread, not trusted yet"
	msgReleaseUntrusted = "app release not trusted"
	msgCacheUnread      = "cache of trusted app releases unread, starting without trusted releases"
	msgCacheUnwritten   = "cache of trusted app releases not written"
)

type RefreshLog struct{}

func (RefreshLog) Refreshed(ctx context.Context, trustedReleases int) {
	slog.DebugContext(ctx, msgRefreshed, slog.Int("trustedReleases", trustedReleases))
}

func (RefreshLog) DenialsUnread(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgDenialsUnread, slog.Any("error", cause))
}

func (RefreshLog) ReleasesUnlisted(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgReleasesUnlisted, slog.Any("error", cause))
}

func (RefreshLog) ReleaseUnread(ctx context.Context, tag string, cause error) {
	slog.WarnContext(ctx, msgReleaseUnread, slog.String("tag", tag), slog.Any("error", cause))
}

func (RefreshLog) ReleaseUntrusted(ctx context.Context, tag string, reason error) {
	slog.WarnContext(ctx, msgReleaseUntrusted, slog.String("tag", tag), slog.Any("reason", reason))
}

func (RefreshLog) CacheUnread(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgCacheUnread, slog.Any("error", cause))
}

func (RefreshLog) CacheUnwritten(ctx context.Context, cause error) {
	slog.WarnContext(ctx, msgCacheUnwritten, slog.Any("error", cause))
}
