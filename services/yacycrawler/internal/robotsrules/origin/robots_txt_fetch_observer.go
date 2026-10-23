package origin

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type RobotsTxtFetchObserver interface {
	RobotsTxtAnswered(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL)
	RobotsTxtUnreachable(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL, cause error)
	RobotsTxtFetchCanceled(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL)
}

type RobotsTxtFetchObservers []RobotsTxtFetchObserver

func (observers RobotsTxtFetchObservers) RobotsTxtAnswered(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.RobotsTxtAnswered(ctx, robotsTxtURL)
	}
}

func (observers RobotsTxtFetchObservers) RobotsTxtUnreachable(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.RobotsTxtUnreachable(ctx, robotsTxtURL, cause)
	}
}

func (observers RobotsTxtFetchObservers) RobotsTxtFetchCanceled(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.RobotsTxtFetchCanceled(ctx, robotsTxtURL)
	}
}
