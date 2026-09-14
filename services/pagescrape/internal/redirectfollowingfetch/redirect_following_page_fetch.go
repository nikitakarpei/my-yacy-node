// Package redirectfollowingfetch fetches a page, follows the redirects the origin
// answers with up to a hop limit, and reports the URL the fetch landed on.
package redirectfollowingfetch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type LandedFetch struct {
	URL     canonicalurl.CanonicalURL
	Outcome pagefetch.FetchOutcome
}

type RedirectFollowingPageFetch struct {
	originPageFetch pagefetch.Fetcher
	maxRedirectHops int
}

func New(originPageFetch pagefetch.Fetcher, maxRedirectHops int) *RedirectFollowingPageFetch {
	return &RedirectFollowingPageFetch{
		originPageFetch: originPageFetch,
		maxRedirectHops: maxRedirectHops,
	}
}

func (f *RedirectFollowingPageFetch) Fetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (LandedFetch, error) {
	fetchURL, version := pageURL, knownVersion
	followedURLs := map[canonicalurl.CanonicalURL]struct{}{}
	for hop := 0; ; hop++ {
		outcome, err := f.originPageFetch.Fetch(ctx, fetchURL, version)
		if err != nil {
			return LandedFetch{}, err
		}
		landed := LandedFetch{URL: fetchURL, Outcome: outcome}
		if outcome.Status != pagefetch.FetchRedirected {
			return landed, nil
		}
		followedURLs[fetchURL] = struct{}{}
		if _, followed := followedURLs[outcome.RedirectTarget]; followed {
			return landed, nil
		}
		if hop == f.maxRedirectHops {
			return landed, nil
		}
		fetchURL, version = outcome.RedirectTarget, pagefetch.PageVersion{}
	}
}
