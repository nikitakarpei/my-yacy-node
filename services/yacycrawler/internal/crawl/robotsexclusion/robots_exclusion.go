// Package robotsexclusion tells whether the robots.txt of a page's site lets the crawler
// fetch that page now, by the group of rules the site states for the crawler's product token.
package robotsexclusion

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const robotsTxtPath = "/robots.txt"

type RobotsRules interface {
	RulesOf(ctx context.Context, robotsTxtURL canonicalurl.CanonicalURL) Rules
}

type RobotsExclusion struct {
	robotsRules  RobotsRules
	productToken ProductToken
}

func New(robotsRules RobotsRules, productToken ProductToken) *RobotsExclusion {
	return &RobotsExclusion{robotsRules: robotsRules, productToken: productToken}
}

func (exclusion *RobotsExclusion) FetchPermissionOf(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) FetchPermission {
	rules := exclusion.robotsRules.RulesOf(ctx, robotsTxtURLOf(pageURL))
	return rules.GroupFor(exclusion.productToken).FetchPermissionOf(pageURL)
}

func robotsTxtURLOf(pageURL canonicalurl.CanonicalURL) canonicalurl.CanonicalURL {
	robotsTxtURL, _ := pageURL.CanonicalURLOfLink(robotsTxtPath)
	return robotsTxtURL
}
