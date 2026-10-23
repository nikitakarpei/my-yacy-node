// Package origin reads the robots.txt rules of a site from the site itself. It follows the
// redirects of robots.txt within one fetch deadline and states how long the rules it read
// stay valid.
package origin

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
)

const (
	RobotsTxtParseLimit   = 500 << 10
	maxRobotsTxtRedirects = 5
)

type RobotsRules struct {
	robotsTxtFetch *redirectfollowingfetch.RedirectFollowingPageFetch
	fetchDeadline  time.Duration
	observer       RobotsTxtFetchObserver
}

func New(
	pageFetch pagefetch.Fetcher,
	fetchDeadline time.Duration,
	observer RobotsTxtFetchObserver,
) *RobotsRules {
	return &RobotsRules{
		robotsTxtFetch: redirectfollowingfetch.New(pageFetch, maxRobotsTxtRedirects),
		fetchDeadline:  fetchDeadline,
		observer:       observer,
	}
}

func (origin *RobotsRules) RulesOf(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) robotsexclusion.Rules {
	fetchOutcome, fetched := origin.fetchWithinDeadline(ctx, robotsTxtURL)
	if !fetched {
		origin.observer.RobotsTxtFetchCanceled(ctx, robotsTxtURL)
		return robotsexclusion.Rules{}
	}
	rules := rulesFromFetch(fetchOutcome)
	if rules.Availability == robotsexclusion.Unreachable {
		origin.observer.RobotsTxtUnreachable(ctx, robotsTxtURL, unreachableCauseOf(fetchOutcome))
		return rules
	}
	origin.observer.RobotsTxtAnswered(ctx, robotsTxtURL)
	return rules
}

func (origin *RobotsRules) fetchWithinDeadline(
	ctx context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) (pagefetch.FetchOutcome, bool) {
	fetchCtx, cancel := context.WithTimeout(ctx, origin.fetchDeadline)
	defer cancel()
	landedFetch, err := origin.robotsTxtFetch.Fetch(fetchCtx, robotsTxtURL, pagefetch.PageVersion{})
	if ctx.Err() != nil {
		return pagefetch.FetchOutcome{}, false
	}
	if err != nil {
		return pagefetch.FetchOutcome{Status: pagefetch.FetchDeadlinePassed}, true
	}
	return landedFetch.Outcome, true
}
