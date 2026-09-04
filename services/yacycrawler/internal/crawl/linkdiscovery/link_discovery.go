// Package linkdiscovery reads the URLs a page's HTML points to.
package linkdiscovery

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

const (
	elementAnchor = "a"
	elementBase   = "base"
)

type LinkDiscovery struct {
	observer LinkResolutionObserver
}

func NewLinkDiscovery(observer LinkResolutionObserver) *LinkDiscovery {
	return &LinkDiscovery{observer: observer}
}

func (discovery *LinkDiscovery) LinkedURLsFrom(
	ctx context.Context,
	elementTree pagehtml.ElementTree,
	pageURL canonicalurl.CanonicalURL,
) []canonicalurl.CanonicalURL {
	baseURL := discovery.baseURLOf(ctx, pageURL, baseHrefOf(elementTree))
	urls, unresolved := distinctURLsFrom(linkHrefsOf(elementTree), baseURL)
	discovery.reportUnresolvedLinkHrefs(ctx, baseURL, unresolved)
	return urls
}

func (discovery *LinkDiscovery) baseURLOf(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	baseHref string,
) canonicalurl.CanonicalURL {
	if baseHref == "" {
		return pageURL
	}
	base, err := pageURL.CanonicalURLOfLink(baseHref)
	if err != nil {
		discovery.observer.BaseHrefUnresolved(ctx, pageURL, baseHref, err)
		return pageURL
	}
	return base
}

func (discovery *LinkDiscovery) reportUnresolvedLinkHrefs(
	ctx context.Context,
	baseURL canonicalurl.CanonicalURL,
	unresolved int,
) {
	if unresolved == 0 {
		return
	}
	discovery.observer.LinkHrefsUnresolved(ctx, baseURL, unresolved)
}

func baseHrefOf(elementTree pagehtml.ElementTree) string {
	for element := range elementTree.ElementsNamed(elementBase) {
		if href, ok := element.AttributeOf("href"); ok {
			return href
		}
	}
	return ""
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
