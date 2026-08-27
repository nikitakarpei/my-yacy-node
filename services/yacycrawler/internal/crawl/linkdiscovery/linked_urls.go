// Package linkdiscovery reads the URLs a page's HTML points to.
package linkdiscovery

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

const (
	elementAnchor = "a"
	elementBase   = "base"

	msgBaseHrefUnresolved  = "base href unresolved, using page url"
	msgLinkHrefsUnresolved = "link hrefs unresolved, left off the frontier"
)

func LinkedURLsFrom(
	ctx context.Context,
	elementTree pagehtml.ElementTree,
	pageURL canonicalurl.CanonicalURL,
) []canonicalurl.CanonicalURL {
	baseURL := baseURLOf(ctx, pageURL, baseHrefOf(elementTree))
	urls, unresolved := distinctURLsFrom(linkHrefsOf(elementTree), baseURL)
	reportUnresolvedLinkHrefs(ctx, baseURL, unresolved)
	return urls
}

func baseHrefOf(elementTree pagehtml.ElementTree) string {
	for element := range elementTree.ElementsNamed(elementBase) {
		if href, ok := element.AttributeOf("href"); ok {
			return href
		}
	}
	return ""
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
		slog.WarnContext(ctx, msgBaseHrefUnresolved,
			slog.String("url", pageURL.String()),
			slog.String("baseHref", baseHref),
			slog.Any("error", err),
		)
		return pageURL
	}
	return base
}

func linkHrefsOf(elementTree pagehtml.ElementTree) []string {
	var hrefs []string
	for element := range elementTree.ElementsNamed(elementAnchor) {
		if href, ok := element.AttributeOf("href"); ok {
			hrefs = append(hrefs, href)
		}
	}
	return hrefs
}

func distinctURLsFrom(
	hrefs []string,
	baseURL canonicalurl.CanonicalURL,
) ([]canonicalurl.CanonicalURL, int) {
	var urls []canonicalurl.CanonicalURL
	unresolved := 0
	seen := map[canonicalurl.CanonicalURL]struct{}{}
	for _, href := range hrefs {
		canonical, err := baseURL.CanonicalURLOfLink(href)
		if err != nil {
			unresolved++
			continue
		}
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		urls = append(urls, canonical)
	}
	return urls, unresolved
}

func reportUnresolvedLinkHrefs(
	ctx context.Context,
	baseURL canonicalurl.CanonicalURL,
	unresolved int,
) {
	if unresolved == 0 {
		return
	}
	slog.WarnContext(ctx, msgLinkHrefsUnresolved,
		slog.String("url", baseURL.String()),
		slog.Int("hrefs", unresolved),
	)
}
