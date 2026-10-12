package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgRobotsTxtAnswered      = "robots.txt answered"
	msgRobotsTxtUnreachable   = "robots.txt unreachable, pages of its site wait"
	msgRobotsTxtFetchCanceled = "robots.txt fetch canceled"
)

type RobotsTxtFetchLog struct{}

func (RobotsTxtFetchLog) RobotsTxtAnswered(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgRobotsTxtAnswered, slog.String("url", robotsTxtURL.String()))
}

func (RobotsTxtFetchLog) RobotsTxtUnreachable(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgRobotsTxtUnreachable,
		slog.String("url", robotsTxtURL.String()),
		slog.Any("error", cause),
	)
}

func (RobotsTxtFetchLog) RobotsTxtFetchCanceled(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgRobotsTxtFetchCanceled, slog.String("url", robotsTxtURL.String()))
}
