package documentextraction

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgBaseHrefUnresolved = "base href unresolved, using page url"

type linkCounts struct {
	local    int
	external int
}

func linkCountsOf(baseURL canonicalurl.CanonicalURL, hrefs []string) linkCounts {
	baseHost := baseURL.Hostname()
	var counts linkCounts
	seen := map[canonicalurl.CanonicalURL]struct{}{}
	for _, href := range hrefs {
		canonical, err := baseURL.CanonicalURLOfLink(href)
		if err != nil {
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		if canonical.Hostname() == baseHost {
			counts.local++
		} else {
			counts.external++
		}
	}
	return counts
}

func baseURLOf(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	baseHref string,
) canonicalurl.CanonicalURL {
	if baseHref == "" {
		return pageURL
	}
	base, err := pageURL.CanonicalURLOfLink(baseHref)
	if err != nil {
		slog.DebugContext(ctx, msgBaseHrefUnresolved,
			slog.String("url", pageURL.String()),
			slog.String("baseHref", baseHref),
			slog.Any("error", err),
		)
		return pageURL
	}
	return base
}
