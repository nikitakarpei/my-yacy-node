package trustedappreleases

import "context"

type ReleaseSource interface {
	TagsOldestFirst(ctx context.Context) ([]string, error)
	ReleaseOf(ctx context.Context, tag string) (Release, error)
}
