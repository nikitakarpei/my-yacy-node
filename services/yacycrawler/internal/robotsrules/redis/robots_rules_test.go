package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotsrules/redis"
)

const robotsTxtURL = "http://example.com/robots.txt"

type siteRobotsRules struct {
	rules robotsexclusion.Rules
	asked int
}

func (site *siteRobotsRules) RulesOf(
	context.Context, canonicalurl.CanonicalURL,
) robotsexclusion.Rules {
	site.asked++
	return site.rules
}

type recordedRobotsRules struct {
	found, missed int
	failures      []error
}

func (recorded *recordedRobotsRules) RobotsRulesFound(context.Context, canonicalurl.CanonicalURL) {
	recorded.found++
}

func (recorded *recordedRobotsRules) RobotsRulesMissed(context.Context, canonicalurl.CanonicalURL) {
	recorded.missed++
}

func (recorded *recordedRobotsRules) RobotsRulesNotRead(
	_ context.Context, _ canonicalurl.CanonicalURL, cause error,
) {
	recorded.failures = append(recorded.failures, cause)
}

func (recorded *recordedRobotsRules) RobotsRulesNotKept(
	_ context.Context, _ canonicalurl.CanonicalURL, cause error,
) {
	recorded.failures = append(recorded.failures, cause)
}

type keptRulesFixture struct {
	server   *miniredis.Miniredis
	site     *siteRobotsRules
	recorded *recordedRobotsRules
	kept     *redis.RobotsRules
}

func keptRulesOn(server *miniredis.Miniredis, rules robotsexclusion.Rules) *keptRulesFixture {
	fixture := &keptRulesFixture{
		server:   server,
		site:     &siteRobotsRules{rules: rules},
		recorded: &recordedRobotsRules{},
	}
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	fixture.kept = redis.New(client, fixture.site, fixture.recorded)
	return fixture
}

func (fixture *keptRulesFixture) rulesOf(t *testing.T) robotsexclusion.Rules {
	t.Helper()
	return fixture.kept.RulesOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, robotsTxtURL),
	)
}

func statedRules() robotsexclusion.Rules {
	return robotsexclusion.Rules{
		Availability: robotsexclusion.Stated,
		RobotsTxt:    "User-agent: *\nDisallow: /\n",
		Lifetime:     time.Hour,
	}
}

func TestRulesAreReadFromTheSiteOnceWhileTheyStayValid(t *testing.T) {
	fixture := keptRulesOn(miniredis.RunT(t), statedRules())

	first := fixture.rulesOf(t)
	second := fixture.rulesOf(t)

	if fixture.site.asked != 1 {
		t.Fatalf("site asked %d times, want once", fixture.site.asked)
	}
	if first != statedRules() || second != statedRules() {
		t.Fatalf("rules = %+v then %+v, want the stated rules twice", first, second)
	}
	if fixture.recorded.missed != 1 || fixture.recorded.found != 1 {
		t.Fatalf("recorded %+v, want one miss then one find", fixture.recorded)
	}
}

func TestRulesPastTheirLifetimeAreReadFromTheSiteAgain(t *testing.T) {
	server := miniredis.RunT(t)
	fixture := keptRulesOn(server, statedRules())

	fixture.rulesOf(t)
	server.FastForward(time.Hour)
	fixture.rulesOf(t)

	if fixture.site.asked != 2 {
		t.Fatalf("site asked %d times, want twice", fixture.site.asked)
	}
}

func TestUnreachableRulesKeepTheirLifetime(t *testing.T) {
	unreachable := robotsexclusion.Rules{
		Availability: robotsexclusion.Unreachable, Lifetime: 2 * time.Minute,
	}
	fixture := keptRulesOn(miniredis.RunT(t), unreachable)

	fixture.rulesOf(t)
	rules := fixture.rulesOf(t)

	if rules != unreachable || fixture.site.asked != 1 {
		t.Fatalf(
			"rules = %+v after %d asks, want the kept unreachable rules",
			rules,
			fixture.site.asked,
		)
	}
}

func TestUndeterminedRulesAreNeverKept(t *testing.T) {
	fixture := keptRulesOn(miniredis.RunT(t), robotsexclusion.Rules{})

	fixture.rulesOf(t)
	fixture.rulesOf(t)

	if fixture.site.asked != 2 {
		t.Fatalf("site asked %d times, want every time", fixture.site.asked)
	}
}

func TestRulesTheServerRefusesAreReadFromTheSiteAndReported(t *testing.T) {
	server := miniredis.RunT(t)
	server.SetError("server down")
	fixture := keptRulesOn(server, statedRules())

	rules := fixture.rulesOf(t)

	if rules != statedRules() {
		t.Fatalf("rules = %+v, want the rules the site states", rules)
	}
	if len(fixture.recorded.failures) != 2 {
		t.Fatalf(
			"reported %v, want the refused read and the refused write",
			fixture.recorded.failures,
		)
	}
}

func TestAnUnreadableRecordIsReadFromTheSiteAgain(t *testing.T) {
	server := miniredis.RunT(t)
	fixture := keptRulesOn(server, statedRules())
	fixture.rulesOf(t)
	for _, key := range server.Keys() {
		if err := server.Set(key, `{"availability":"unknown"}`); err != nil {
			t.Fatal(err)
		}
	}

	fixture.rulesOf(t)

	if fixture.site.asked != 2 {
		t.Fatalf("site asked %d times, want the unreadable record replaced", fixture.site.asked)
	}
	if len(fixture.recorded.failures) != 1 {
		t.Fatalf("reported %v, want the unreadable record", fixture.recorded.failures)
	}
}

func TestARecordThatIsNotJSONIsReadFromTheSiteAgain(t *testing.T) {
	server := miniredis.RunT(t)
	fixture := keptRulesOn(server, statedRules())
	fixture.rulesOf(t)
	for _, key := range server.Keys() {
		if err := server.Set(key, "not json"); err != nil {
			t.Fatal(err)
		}
	}

	fixture.rulesOf(t)

	if fixture.site.asked != 2 || len(fixture.recorded.failures) != 1 {
		t.Fatalf("site asked %d times, reported %v, want a refetch and one report",
			fixture.site.asked, fixture.recorded.failures)
	}
}
