package redis

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type RobotsRulesObserver interface {
	RobotsRulesFound(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL)
	RobotsRulesMissed(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL)
	RobotsRulesNotRead(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL, cause error)
	RobotsRulesNotKept(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL, cause error)
}

type RobotsRulesObservers []RobotsRulesObserver

func (observers RobotsRulesObservers) RobotsRulesFound(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.RobotsRulesFound(ctx, robotsTxtURL)
	}
}

func (observers RobotsRulesObservers) RobotsRulesMissed(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.RobotsRulesMissed(ctx, robotsTxtURL)
	}
}

func (observers RobotsRulesObservers) RobotsRulesNotRead(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.RobotsRulesNotRead(ctx, robotsTxtURL, cause)
	}
}

func (observers RobotsRulesObservers) RobotsRulesNotKept(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.RobotsRulesNotKept(ctx, robotsTxtURL, cause)
	}
}
