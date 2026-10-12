package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgRobotsRulesFound   = "robots rules found"
	msgRobotsRulesMissed  = "robots rules missed, the crawler fetches robots.txt"
	msgRobotsRulesNotRead = "robots rules not read, the crawler fetches robots.txt"
	msgRobotsRulesNotKept = "robots rules not kept, the next page of the site fetches robots.txt again"
)

type RobotsRulesLog struct{}

func (RobotsRulesLog) RobotsRulesFound(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgRobotsRulesFound, slog.String("url", robotsTxtURL.String()))
}

func (RobotsRulesLog) RobotsRulesMissed(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgRobotsRulesMissed, slog.String("url", robotsTxtURL.String()))
}

func (RobotsRulesLog) RobotsRulesNotRead(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgRobotsRulesNotRead,
		slog.String("url", robotsTxtURL.String()),
		slog.Any("error", cause),
	)
}

func (RobotsRulesLog) RobotsRulesNotKept(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgRobotsRulesNotKept,
		slog.String("url", robotsTxtURL.String()),
		slog.Any("error", cause),
	)
}
