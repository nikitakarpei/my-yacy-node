package documentextraction

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgBaseHrefUnresolved = "base href unresolved, using page url"

func distinctLinksFrom(
	hrefs []string,
	baseURL canonicalurl.CanonicalURL,
) []canonicalurl.CanonicalURL {
	links := make([]canonicalurl.CanonicalURL, 0, len(hrefs))
	seen := map[canonicalurl.CanonicalURL]struct{}{}
	for _, href := range hrefs {
		link, err := baseURL.CanonicalURLOfLink(href)
		if err != nil {
			continue
		}
		if _, ok := seen[link]; ok {
			continue
		}
		seen[link] = struct{}{}
		links = append(links, link)
	}
	return links
}

func localLinksOf(links []canonicalurl.CanonicalURL, baseHost string) int {
	localLinks := 0
	for _, link := range links {
		if link.Hostname() == baseHost {
			localLinks++
		}
	}
	return localLinks
}

func externalLinksOf(links []canonicalurl.CanonicalURL, baseHost string) int {
	externalLinks := 0
	for _, link := range links {
		if link.Hostname() != baseHost {
			externalLinks++
		}
	}
	return externalLinks
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
