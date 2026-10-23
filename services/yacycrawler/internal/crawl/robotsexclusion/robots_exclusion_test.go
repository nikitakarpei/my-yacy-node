package robotsexclusion_test

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
)

const productToken = robotsexclusion.ProductToken("yacycrawler")

type stubRobotsRules struct {
	rules              robotsexclusion.Rules
	askedRobotsTxtURLs []string
}

func (stub *stubRobotsRules) RulesOf(
	_ context.Context,
	robotsTxtURL canonicalurl.CanonicalURL,
) robotsexclusion.Rules {
	stub.askedRobotsTxtURLs = append(stub.askedRobotsTxtURLs, robotsTxtURL.String())
	return stub.rules
}

func statedRules(robotsTxt string) robotsexclusion.Rules {
	return robotsexclusion.Rules{Availability: robotsexclusion.Stated, RobotsTxt: robotsTxt}
}

func fetchPermissionOf(
	t *testing.T,
	rules robotsexclusion.Rules,
	pageURL string,
) robotsexclusion.FetchPermission {
	t.Helper()
	exclusion := robotsexclusion.New(&stubRobotsRules{rules: rules}, productToken)
	return exclusion.FetchPermissionOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageURL),
	)
}

func TestTheRulesOfThePageSiteAreAsked(t *testing.T) {
	robotsRules := &stubRobotsRules{rules: robotsexclusion.Rules{
		Availability: robotsexclusion.Absent,
	}}
	exclusion := robotsexclusion.New(robotsRules, productToken)

	exclusion.FetchPermissionOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "https://Example.com:8443/a/b.html?q=1#top"),
	)

	want := "https://example.com:8443/robots.txt"
	if len(robotsRules.askedRobotsTxtURLs) != 1 || robotsRules.askedRobotsTxtURLs[0] != want {
		t.Fatalf("asked rules of %v, want %s", robotsRules.askedRobotsTxtURLs, want)
	}
}

func TestAPageTheGroupOfTheProductTokenDisallowsIsDisallowed(t *testing.T) {
	rules := statedRules("User-agent: yacycrawler\nDisallow: /private/\n")

	permission := fetchPermissionOf(t, rules, "http://example.com/private/a.html")

	if permission.Verdict != robotsexclusion.Disallowed {
		t.Fatalf("verdict = %v, want disallowed", permission.Verdict)
	}
}

func TestAPageOutsideEveryDisallowedPathIsAllowed(t *testing.T) {
	rules := statedRules("User-agent: yacycrawler\nDisallow: /private/\n")

	permission := fetchPermissionOf(t, rules, "http://example.com/public/a.html")

	if permission.Verdict != robotsexclusion.Allowed {
		t.Fatalf("verdict = %v, want allowed", permission.Verdict)
	}
}

func TestTheLongestMatchingRuleDecides(t *testing.T) {
	rules := statedRules(
		"User-agent: *\nDisallow: /private/\nAllow: /private/public/\n",
	)

	permission := fetchPermissionOf(t, rules, "http://example.com/private/public/a.html")

	if permission.Verdict != robotsexclusion.Allowed {
		t.Fatalf("verdict = %v, want the longer allow rule to win", permission.Verdict)
	}
}

func TestTheGroupOfTheProductTokenReplacesTheGroupOfEveryCrawler(t *testing.T) {
	rules := statedRules(
		"User-agent: *\nDisallow: /\n\nUser-agent: YaCyCrawler\nDisallow: /private/\n",
	)

	permission := fetchPermissionOf(t, rules, "http://example.com/a.html")

	if permission.Verdict != robotsexclusion.Allowed {
		t.Fatalf("verdict = %v, want the group of the product token to apply", permission.Verdict)
	}
}

func TestTheGroupOfAnotherCrawlerDoesNotApply(t *testing.T) {
	rules := statedRules("User-agent: othercrawler\nDisallow: /\n")

	permission := fetchPermissionOf(t, rules, "http://example.com/a.html")

	if permission.Verdict != robotsexclusion.Allowed {
		t.Fatalf("verdict = %v, want allowed", permission.Verdict)
	}
}

func TestAnAbsentRobotsTxtAllowsEveryPage(t *testing.T) {
	permission := fetchPermissionOf(
		t,
		robotsexclusion.Rules{Availability: robotsexclusion.Absent},
		"http://example.com/private/a.html",
	)

	if permission.Verdict != robotsexclusion.Allowed {
		t.Fatalf("verdict = %v, want allowed", permission.Verdict)
	}
}

func TestAnUnreachableRobotsTxtDefersThePageForTheLifetimeOfItsRules(t *testing.T) {
	permission := fetchPermissionOf(
		t,
		robotsexclusion.Rules{Availability: robotsexclusion.Unreachable, Lifetime: 5 * time.Minute},
		"http://example.com/a.html",
	)

	if permission.Verdict != robotsexclusion.Deferred || permission.DeferFor != 5*time.Minute {
		t.Fatalf("permission = %+v, want deferred for 5m", permission)
	}
}

func TestUndeterminedRulesDeferThePage(t *testing.T) {
	permission := fetchPermissionOf(t, robotsexclusion.Rules{}, "http://example.com/a.html")

	if permission.Verdict != robotsexclusion.Deferred || permission.DeferFor <= 0 {
		t.Fatalf("permission = %+v, want deferred for a while", permission)
	}
}
