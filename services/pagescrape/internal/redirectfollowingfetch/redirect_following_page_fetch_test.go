package redirectfollowingfetch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/internal/redirectfollowingfetch"
)

const requestedURL = "http://host/a"

var errOriginUnreachable = errors.New("the origin is unreachable")

type originPageFetches struct {
	outcomeByURL map[string]pagefetch.FetchOutcome
	err          error
	fetchedURLs  []string
	sentVersions []pagefetch.PageVersion
}

func (f *originPageFetches) Fetch(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	if f.err != nil {
		return pagefetch.FetchOutcome{}, f.err
	}
	f.fetchedURLs = append(f.fetchedURLs, pageURL.String())
	f.sentVersions = append(f.sentVersions, knownVersion)
	return f.outcomeByURL[pageURL.String()], nil
}

func redirectTo(t *testing.T, targetURL string) pagefetch.FetchOutcome {
	t.Helper()
	return pagefetch.FetchOutcome{
		Status:         pagefetch.FetchRedirected,
		RedirectTarget: canonicalurltest.CanonicalURLOf(t, targetURL),
	}
}

func servedPage(body string) pagefetch.FetchOutcome {
	return pagefetch.FetchOutcome{
		Status: pagefetch.FetchSucceeded,
		Page:   pagefetch.FetchedPage{ContentType: "text/html", Body: []byte(body)},
	}
}

func fetchFollowingRedirects(
	t *testing.T,
	origin *originPageFetches,
	maxRedirectHops int,
) redirectfollowingfetch.LandedFetch {
	t.Helper()
	landed, err := redirectfollowingfetch.New(origin, maxRedirectHops).Fetch(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, requestedURL),
		pagefetch.PageVersion{},
	)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	return landed
}

func TestFetchLandsOnThePageTheRedirectsLeadTo(t *testing.T) {
	origin := &originPageFetches{outcomeByURL: map[string]pagefetch.FetchOutcome{
		"http://host/a": redirectTo(t, "http://host/b"),
		"http://host/b": redirectTo(t, "http://host/c"),
		"http://host/c": servedPage("<html>landed</html>"),
	}}

	landed := fetchFollowingRedirects(t, origin, 10)

	if landed.URL != canonicalurltest.CanonicalURLOf(t, "http://host/c") {
		t.Errorf("landed on %s, want http://host/c", landed.URL)
	}
	if landed.Outcome.Status != pagefetch.FetchSucceeded {
		t.Errorf("status = %v, want succeeded", landed.Outcome.Status)
	}
	if string(landed.Outcome.Page.Body) != "<html>landed</html>" {
		t.Errorf("body = %q", landed.Outcome.Page.Body)
	}
}

func TestFetchOfAPageThatDoesNotRedirectAsksTheOriginOnce(t *testing.T) {
	origin := &originPageFetches{outcomeByURL: map[string]pagefetch.FetchOutcome{
		"http://host/a": servedPage("<html>here</html>"),
	}}

	landed := fetchFollowingRedirects(t, origin, 10)

	if landed.URL != canonicalurltest.CanonicalURLOf(t, "http://host/a") {
		t.Errorf("landed on %s, want http://host/a", landed.URL)
	}
	if len(origin.fetchedURLs) != 1 {
		t.Errorf("asked the origin %v, want only http://host/a", origin.fetchedURLs)
	}
}

func TestFetchSendsTheKnownPageVersionOnlyToTheURLItNames(t *testing.T) {
	origin := &originPageFetches{outcomeByURL: map[string]pagefetch.FetchOutcome{
		"http://host/a": redirectTo(t, "http://host/b"),
		"http://host/b": servedPage("<html>landed</html>"),
	}}
	knownVersion := pagefetch.PageVersion{EntityTag: "\"known\""}

	if _, err := redirectfollowingfetch.New(origin, 10).Fetch(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/a"),
		knownVersion,
	); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if origin.sentVersions[0] != knownVersion {
		t.Errorf("sent %+v to the named page, want %+v", origin.sentVersions[0], knownVersion)
	}
	if origin.sentVersions[1] != (pagefetch.PageVersion{}) {
		t.Errorf("sent %+v to the redirect target, want no version",
			origin.sentVersions[1])
	}
}

func TestFetchStopsAtTheHopLimitAndReportsTheRedirect(t *testing.T) {
	origin := &originPageFetches{outcomeByURL: map[string]pagefetch.FetchOutcome{
		"http://host/a": redirectTo(t, "http://host/b"),
		"http://host/b": redirectTo(t, "http://host/c"),
		"http://host/c": servedPage("<html>landed</html>"),
	}}

	landed := fetchFollowingRedirects(t, origin, 1)

	if landed.Outcome.Status != pagefetch.FetchRedirected {
		t.Errorf("status = %v, want redirected", landed.Outcome.Status)
	}
	if landed.URL != canonicalurltest.CanonicalURLOf(t, "http://host/b") {
		t.Errorf("stopped at %s, want http://host/b", landed.URL)
	}
}

func TestFetchStopsWhenTheRedirectsCircleBack(t *testing.T) {
	origin := &originPageFetches{outcomeByURL: map[string]pagefetch.FetchOutcome{
		"http://host/a": redirectTo(t, "http://host/b"),
		"http://host/b": redirectTo(t, "http://host/a"),
	}}

	landed := fetchFollowingRedirects(t, origin, 10)

	if landed.Outcome.Status != pagefetch.FetchRedirected {
		t.Errorf("status = %v, want redirected", landed.Outcome.Status)
	}
	if len(origin.fetchedURLs) != 2 {
		t.Errorf("asked the origin %v, want each URL once", origin.fetchedURLs)
	}
}

func TestFetchReportsWhatTheOriginFetchFailedWith(t *testing.T) {
	origin := &originPageFetches{err: errOriginUnreachable}

	_, err := redirectfollowingfetch.New(origin, 10).Fetch(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://host/a"),
		pagefetch.PageVersion{},
	)

	if !errors.Is(err, errOriginUnreachable) {
		t.Errorf("error = %v, want %v", err, errOriginUnreachable)
	}
}
