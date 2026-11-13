package trustedappreleases

import "context"

type RefreshObserver interface {
	Refreshed(ctx context.Context, trustedReleases int)
	DenialsUnread(ctx context.Context, cause error)
	ReleasesUnlisted(ctx context.Context, cause error)
	ReleaseUnread(ctx context.Context, tag string, cause error)
	ReleaseUntrusted(ctx context.Context, tag string, reason error)
	CacheUnread(ctx context.Context, cause error)
	CacheUnwritten(ctx context.Context, cause error)
}
