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

const pageURL = "https://example.com/page"

var (
	errCorpusUnreachable = errors.New("corpus unreachable")
	storedAt             = time.Date(2026, time.August, 25, 10, 30, 0, 0, time.UTC)
)

type heldMarkdown struct {
	byCanonicalURL map[string]string
	failure        error
}

func (h heldMarkdown) MarkdownOf(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (markdownrecall.StoredMarkdown, bool, error) {
	if h.failure != nil {
		return markdownrecall.StoredMarkdown{}, false, h.failure
	}
	markdown, held := h.byCanonicalURL[canonicalURL.String()]
	if !held {
		return markdownrecall.StoredMarkdown{}, false, nil
	}
	return markdownrecall.StoredMarkdown{
		Markdown: []byte(markdown),
		StoredAt: storedAt,
		Version:  versionOf(markdown),
	}, true, nil
}

func versionOf(markdown string) string {
	return "version-of-" + markdown
}

func pageOf(
	t *testing.T,
	corpus markdownrecall.PageMarkdownCorpus,
	url string,
) (markdownrecall.RecalledPage, bool, error) {
	t.Helper()
	return markdownrecall.NewPageMarkdownRecall(corpus).PageOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, url),
	)
}

func TestPageOfYieldsTheMarkdownHeldUnderTheRequestedURL(t *testing.T) {
	page, held, err := pageOf(t,
		heldMarkdown{byCanonicalURL: map[string]string{pageURL: "# Hi"}},
		pageURL,
	)
	if err != nil {
		t.Fatalf("page of %q: %v", pageURL, err)
	}
	if !held {
		t.Fatalf("no page held for %q", pageURL)
	}
	if string(page.Markdown) != "# Hi" {
		t.Errorf("markdown = %q, want %q", page.Markdown, "# Hi")
	}
	if page.MarkdownURL.String() != pageURL {
		t.Errorf("markdownURL = %q, want %q", page.MarkdownURL, pageURL)
	}
	if page.StoredAt != storedAt {
		t.Errorf("storedAt = %v, want %v", page.StoredAt, storedAt)
	}
	if page.Version != versionOf("# Hi") {
		t.Errorf("version = %q, want %q", page.Version, versionOf("# Hi"))
	}
}

func TestPageOfHoldsNoPageForAURLTheCorpusNeverStored(t *testing.T) {
	_, held, err := pageOf(t, heldMarkdown{}, pageURL)
	if err != nil {
		t.Fatalf("page of %q: %v", pageURL, err)
	}
	if held {
		t.Fatal("a page was held for a url the corpus never stored")
	}
}

func TestPageOfPassesTheCorpusFailureThrough(t *testing.T) {
	_, _, err := pageOf(t, heldMarkdown{failure: errCorpusUnreachable}, pageURL)
	if !errors.Is(err, errCorpusUnreachable) {
		t.Fatalf("err = %v, want %v", err, errCorpusUnreachable)
	}
}
