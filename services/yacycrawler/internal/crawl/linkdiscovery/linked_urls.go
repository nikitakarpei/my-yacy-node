// Package linkdiscovery reads the URLs a page's markup points to.
package linkdiscovery

import (
	"context"
	"log/slog"

	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagemarkup"
)

const msgBaseHrefUnresolved = "base href unresolved, using page url"

func LinkedURLsFrom(
	ctx context.Context,
	markup pagemarkup.Markup,
	pageURL canonicalurl.CanonicalURL,
) []canonicalurl.CanonicalURL {
	return distinctURLsFrom(
		linkHrefsOf(markup),
		baseURLOf(ctx, pageURL, baseHrefOf(markup)),
	)
}

func linkHrefsOf(markup pagemarkup.Markup) []string {
	var hrefs []string
	for node := range markup.Elements() {
		if node.DataAtom != atom.A {
			continue
		}
		if href, ok := pagemarkup.AttributeOf(node, "href"); ok {
			hrefs = append(hrefs, href)
		}
	}
	return hrefs
}

func baseHrefOf(markup pagemarkup.Markup) string {
	for node := range markup.Elements() {
		if node.DataAtom != atom.Base {
			continue
		}
		if href, ok := pagemarkup.AttributeOf(node, "href"); ok {
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
		slog.DebugContext(ctx, msgBaseHrefUnresolved,
			slog.String("url", pageURL.String()),
			slog.String("baseHref", baseHref),
			slog.Any("error", err),
		)
		return pageURL
	}
	return base
}

func distinctURLsFrom(
	hrefs []string,
	baseURL canonicalurl.CanonicalURL,
) []canonicalurl.CanonicalURL {
	var urls []canonicalurl.CanonicalURL
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
		urls = append(urls, canonical)
	}
	return urls
}
