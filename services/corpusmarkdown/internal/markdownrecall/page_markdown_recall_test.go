package markdownrecall_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
)

const (
	requestedURL = "http://example.com/page"
	settledURL   = "https://example.com/page"
)

var (
	errCorpusUnreachable       = errors.New("corpus unreachable")
	errRedirectionsUnreachable = errors.New("redirections unreachable")
	storedAt                   = time.Date(2026, time.August, 25, 10, 30, 0, 0, time.UTC)
)

type heldMarkdown struct {
	byCanonicalURL map[string]string
	failure        error
}

func (h heldMarkdown) MarkdownOf(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) ([]byte, time.Time, bool, error) {
	if h.failure != nil {
		return nil, time.Time{}, false, h.failure
	}
	markdown, held := h.byCanonicalURL[canonicalURL.String()]
	if !held {
		return nil, time.Time{}, false, nil
	}
	return []byte(markdown), storedAt, true, nil
}

type recordedRedirections struct {
	byRequestedURL map[string]string
	failure        error
}

func (r recordedRedirections) RedirectionOf(
	_ context.Context,
	url canonicalurl.CanonicalURL,
) (canonicalurl.CanonicalURL, bool, error) {
	if r.failure != nil {
		return canonicalurl.CanonicalURL{}, false, r.failure
	}
	settled, redirected := r.byRequestedURL[url.String()]
	if !redirected {
		return canonicalurl.CanonicalURL{}, false, nil
	}
	markdownURL, err := canonicalurl.CanonicalURLOf(settled)
	if err != nil {
		return canonicalurl.CanonicalURL{}, false, err
	}
	return markdownURL, true, nil
}

func pageOf(
	t *testing.T,
	corpus markdownrecall.PageMarkdownCorpus,
	redirections markdownrecall.PageRedirections,
	url string,
) (markdownrecall.RecalledPage, bool, error) {
	t.Helper()
	return markdownrecall.NewPageMarkdownRecall(corpus, redirections).PageOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, url),
	)
}

func TestPageOfYieldsTheMarkdownHeldUnderTheRequestedURL(t *testing.T) {
	page, held, err := pageOf(t,
		heldMarkdown{byCanonicalURL: map[string]string{settledURL: "# Hi"}},
		recordedRedirections{},
		settledURL,
	)
	if err != nil {
		t.Fatalf("page of %q: %v", settledURL, err)
	}
	if !held {
		t.Fatalf("no page held for %q", settledURL)
	}
	if string(page.Markdown) != "# Hi" {
		t.Errorf("markdown = %q, want %q", page.Markdown, "# Hi")
	}
	if page.MarkdownURL.String() != settledURL {
		t.Errorf("markdownURL = %q, want %q", page.MarkdownURL, settledURL)
	}
	if page.StoredAt != storedAt {
		t.Errorf("storedAt = %v, want %v", page.StoredAt, storedAt)
	}
}

func TestPageOfFollowsTheRedirectionARequestedURLRecorded(t *testing.T) {
	page, held, err := pageOf(t,
		heldMarkdown{byCanonicalURL: map[string]string{settledURL: "# Hi"}},
		recordedRedirections{byRequestedURL: map[string]string{requestedURL: settledURL}},
		requestedURL,
	)
	if err != nil {
		t.Fatalf("page of %q: %v", requestedURL, err)
	}
	if !held {
		t.Fatalf("no page held for %q, want the page its redirection leads to", requestedURL)
	}
	if page.MarkdownURL.String() != settledURL {
		t.Errorf("markdownURL = %q, want %q", page.MarkdownURL, settledURL)
	}
	if string(page.Markdown) != "# Hi" {
		t.Errorf("markdown = %q, want %q", page.Markdown, "# Hi")
	}
}

func TestPageOfHoldsNoPageForAURLWithNeitherMarkdownNorRedirection(t *testing.T) {
	_, held, err := pageOf(t, heldMarkdown{}, recordedRedirections{}, requestedURL)
	if err != nil {
		t.Fatalf("page of %q: %v", requestedURL, err)
	}
	if held {
		t.Fatal("a page was held for a url the corpus never stored")
	}
}

func TestPageOfHoldsNoPageWhenTheRedirectionLeadsToAnEmptyCorpus(t *testing.T) {
	_, held, err := pageOf(t,
		heldMarkdown{},
		recordedRedirections{byRequestedURL: map[string]string{requestedURL: settledURL}},
		requestedURL,
	)
	if err != nil {
		t.Fatalf("page of %q: %v", requestedURL, err)
	}
	if held {
		t.Fatal("a page was held for a redirection whose target the corpus never stored")
	}
}

func TestPageOfPassesTheCorpusFailureThrough(t *testing.T) {
	_, _, err := pageOf(t,
		heldMarkdown{failure: errCorpusUnreachable}, recordedRedirections{}, requestedURL,
	)
	if !errors.Is(err, errCorpusUnreachable) {
		t.Fatalf("err = %v, want %v", err, errCorpusUnreachable)
	}
}

func TestPageOfPassesTheRedirectionFailureThrough(t *testing.T) {
	_, _, err := pageOf(t,
		heldMarkdown{}, recordedRedirections{failure: errRedirectionsUnreachable}, requestedURL,
	)
	if !errors.Is(err, errRedirectionsUnreachable) {
		t.Fatalf("err = %v, want %v", err, errRedirectionsUnreachable)
	}
}
