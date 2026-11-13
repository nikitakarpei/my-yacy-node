package trustedappreleases

import "context"

type ReleaseSource interface {
	Tags(ctx context.Context) ([]string, error)
	ReleaseOf(ctx context.Context, tag string) (Release, error)
}
