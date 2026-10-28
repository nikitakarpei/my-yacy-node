// Package linkresolution resolves the links a page states into the distinct URLs they
// point to. A stated base URL or a link it cannot resolve goes to the observer.
package linkresolution

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Link struct {
	BaseURL   canonicalurl.CanonicalURL
	Reference string
}

type LinkResolution struct {
	observer LinkResolutionObserver
}

func New(observer LinkResolutionObserver) *LinkResolution {
	return &LinkResolution{observer: observer}
}

func (resolution *LinkResolution) BaseURLOf(
	ctx context.Context,
	enclosingBaseURL canonicalurl.CanonicalURL,
	statedBaseURL string,
) canonicalurl.CanonicalURL {
	if statedBaseURL == "" {
		return enclosingBaseURL
	}
	baseURL, err := enclosingBaseURL.CanonicalURLOfLink(statedBaseURL)
	if err != nil {
		resolution.observer.BaseURLUnresolved(ctx, enclosingBaseURL, statedBaseURL, err)
		return enclosingBaseURL
	}
	return baseURL
}

func (resolution *LinkResolution) URLsOf(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	links []Link,
) []canonicalurl.CanonicalURL {
	urls, unresolved := distinctURLsOf(links)
	if unresolved > 0 {
		resolution.observer.LinksUnresolved(ctx, pageURL, unresolved)
	}
	return urls
}

func distinctURLsOf(links []Link) ([]canonicalurl.CanonicalURL, int) {
	var urls []canonicalurl.CanonicalURL
	unresolved := 0
	seen := map[canonicalurl.CanonicalURL]struct{}{}
	for _, link := range links {
		url, err := link.BaseURL.CanonicalURLOfLink(link.Reference)
		if err != nil {
			unresolved++
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		urls = append(urls, url)
	}
	return urls, unresolved
}
