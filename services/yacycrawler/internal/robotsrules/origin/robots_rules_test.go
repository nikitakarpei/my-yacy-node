package origin_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/robotsexclusion"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/robotsrules/origin"
)

const (
	robotsTxtURL  = "http://example.com/robots.txt"
	fetchDeadline = time.Minute
)

type sitePages map[string]pagefetch.FetchOutcome

func (pages sitePages) Fetch(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	return pages[pageURL.String()], nil
}

type stalledSite struct{}

func (stalledSite) Fetch(
	ctx context.Context,
	_ canonicalurl.CanonicalURL,
	_ pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	<-ctx.Done()
	return pagefetch.FetchOutcome{}, fmt.Errorf("fetch: %w", ctx.Err())
}

type recordedRobotsTxtFetches struct {
	answered          int
	unreachableCauses []error
	canceled          int
}

func (fetches *recordedRobotsTxtFetches) RobotsTxtAnswered(
	context.Context, canonicalurl.CanonicalURL,
) {
	fetches.answered++
}

func (fetches *recordedRobotsTxtFetches) RobotsTxtUnreachable(
	_ context.Context, _ canonicalurl.CanonicalURL, cause error,
) {
	fetches.unreachableCauses = append(fetches.unreachableCauses, cause)
}

func (fetches *recordedRobotsTxtFetches) RobotsTxtFetchCanceled(
	context.Context, canonicalurl.CanonicalURL,
) {
	fetches.canceled++
}

func rulesFrom(
	t *testing.T,
	site pagefetch.Fetcher,
	deadline time.Duration,
) (robotsexclusion.Rules, *recordedRobotsTxtFetches) {
	t.Helper()
	fetches := &recordedRobotsTxtFetches{}
	rules := origin.New(site, deadline, fetches).RulesOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, robotsTxtURL),
	)
	return rules, fetches
}

func rulesOfAnswer(t *testing.T, answer pagefetch.FetchOutcome) robotsexclusion.Rules {
	t.Helper()
	rules, _ := rulesFrom(t, sitePages{robotsTxtURL: answer}, fetchDeadline)
	return rules
}

func TestAFetchedRobotsTxtStatesItsRulesForADay(t *testing.T) {
	rules, fetches := rulesFrom(t, sitePages{robotsTxtURL: {
		Status: pagefetch.FetchSucceeded,
		Page:   pagefetch.FetchedPage{Body: []byte("User-agent: *\nDisallow: /\n")},
	}}, fetchDeadline)

	if rules.Availability != robotsexclusion.Stated ||
		rules.RobotsTxt != "User-agent: *\nDisallow: /\n" {
		t.Fatalf("rules = %+v, want the stated robots.txt", rules)
	}
	if rules.Lifetime != 24*time.Hour {
		t.Fatalf("lifetime = %v, want a day", rules.Lifetime)
	}
	if fetches.answered != 1 {
		t.Fatalf("answered fetches = %d, want 1", fetches.answered)
	}
}

func TestARobotsTxtTheSiteRefusesOrLacksIsAbsentForADay(t *testing.T) {
	for _, status := range []pagefetch.FetchStatus{
		pagefetch.FetchAccessRefused,
		pagefetch.FetchGone,
		pagefetch.FetchRejected,
		pagefetch.FetchOversized,
		pagefetch.FetchRedirectTargetInvalid,
	} {
		rules := rulesOfAnswer(t, pagefetch.FetchOutcome{Status: status})
		if rules.Availability != robotsexclusion.Absent || rules.Lifetime != 24*time.Hour {
			t.Errorf("status %d gave %+v, want absent for a day", status, rules)
		}
	}
}

func TestARedirectedRobotsTxtIsReadWhereItLands(t *testing.T) {
	rules, _ := rulesFrom(t, sitePages{
		robotsTxtURL: {
			Status: pagefetch.FetchRedirected,
			RedirectTarget: canonicalurltest.CanonicalURLOf(
				t,
				"https://www.example.com/robots.txt",
			),
		},
		"https://www.example.com/robots.txt": {
			Status: pagefetch.FetchSucceeded,
			Page:   pagefetch.FetchedPage{Body: []byte("User-agent: *\nDisallow: /a\n")},
		},
	}, fetchDeadline)

	if rules.Availability != robotsexclusion.Stated ||
		rules.RobotsTxt != "User-agent: *\nDisallow: /a\n" {
		t.Fatalf("rules = %+v, want the robots.txt the redirect landed on", rules)
	}
}

func TestARobotsTxtPastFiveRedirectsIsAbsent(t *testing.T) {
	pages := sitePages{}
	for hop := range 7 {
		pages[hopURL(hop)] = pagefetch.FetchOutcome{
			Status:         pagefetch.FetchRedirected,
			RedirectTarget: canonicalurltest.CanonicalURLOf(t, hopURL(hop+1)),
		}
		pages[hopURL(hop+1)] = pagefetch.FetchOutcome{Status: pagefetch.FetchSucceeded}
	}
	pages[robotsTxtURL] = pages[hopURL(0)]

	rules, _ := rulesFrom(t, pages, fetchDeadline)

	if rules.Availability != robotsexclusion.Absent {
		t.Fatalf("rules = %+v, want absent", rules)
	}
}

func hopURL(hop int) string {
	return fmt.Sprintf("http://example.com/hop/%d", hop)
}

func TestARobotsTxtTheSiteAsksToWaitForIsUnreachableWithinAnHour(t *testing.T) {
	for wait, want := range map[time.Duration]time.Duration{
		0:                time.Minute,
		10 * time.Minute: 10 * time.Minute,
		48 * time.Hour:   time.Hour,
	} {
		rules := rulesOfAnswer(t, pagefetch.FetchOutcome{
			Status: pagefetch.FetchDeferred, DeferFor: wait,
		})
		if rules.Availability != robotsexclusion.Unreachable || rules.Lifetime != want {
			t.Errorf("wait %v gave %+v, want unreachable for %v", wait, rules, want)
		}
	}
}

func TestAFailedRobotsTxtFetchIsUnreachableAndReportsItsCause(t *testing.T) {
	cause := errors.New("connection refused")

	rules, fetches := rulesFrom(t, sitePages{robotsTxtURL: {
		Status: pagefetch.FetchFailed, FailureCause: cause,
	}}, fetchDeadline)

	if rules.Availability != robotsexclusion.Unreachable || rules.Lifetime != time.Minute {
		t.Fatalf("rules = %+v, want unreachable for a minute", rules)
	}
	if len(fetches.unreachableCauses) != 1 || !errors.Is(fetches.unreachableCauses[0], cause) {
		t.Fatalf("reported %v, want the failure cause", fetches.unreachableCauses)
	}
}

func TestEveryUnreachableAnswerReportsACause(t *testing.T) {
	for _, status := range []pagefetch.FetchStatus{
		pagefetch.FetchDeferred, pagefetch.FetchDeadlinePassed, pagefetch.FetchNotModified,
	} {
		_, fetches := rulesFrom(t, sitePages{robotsTxtURL: {Status: status}}, fetchDeadline)
		if len(fetches.unreachableCauses) != 1 || fetches.unreachableCauses[0] == nil {
			t.Errorf("status %d reported %v, want one cause", status, fetches.unreachableCauses)
		}
	}
}

func TestARobotsTxtPastTheFetchDeadlineIsUnreachable(t *testing.T) {
	rules, fetches := rulesFrom(t, stalledSite{}, time.Nanosecond)

	if rules.Availability != robotsexclusion.Unreachable {
		t.Fatalf("rules = %+v, want unreachable", rules)
	}
	if len(fetches.unreachableCauses) != 1 {
		t.Fatalf("reported %v, want the passed deadline", fetches.unreachableCauses)
	}
}

func TestAFetchTheCallerCancelsLeavesTheRulesUndetermined(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fetches := &recordedRobotsTxtFetches{}

	rules := origin.New(stalledSite{}, fetchDeadline, fetches).RulesOf(
		ctx, canonicalurltest.CanonicalURLOf(t, robotsTxtURL),
	)

	if rules.Availability != robotsexclusion.Undetermined || rules.Lifetime != 0 {
		t.Fatalf("rules = %+v, want undetermined rules nobody keeps", rules)
	}
	if fetches.canceled != 1 || len(fetches.unreachableCauses) != 0 {
		t.Fatalf("reported %+v, want one canceled fetch", fetches)
	}
}
