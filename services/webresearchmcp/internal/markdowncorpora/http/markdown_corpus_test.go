package http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	markdowncorporahttp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/markdowncorpora/http"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

const (
	pageAddress    = "https://example.org/page"
	heldMarkdown   = "# Research subject"
	heldVersion    = "version-1"
	recallDeadline = 5 * time.Second
)

type corpusServing struct {
	held     *pagemarkdownstore.RecalledPage
	notFound bool
}

func (s *corpusServing) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	if s.notFound {
		http.Error(writer, "the corpus holds no markdown for the page", http.StatusNotFound)
		return
	}
	if s.held == nil {
		http.Error(writer, "the corpus is away", http.StatusServiceUnavailable)
		return
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(s.held)
}

func openCorpusUnderTest(t *testing.T, serving *corpusServing) *markdowncorporahttp.MarkdownCorpus {
	t.Helper()
	corpus := httptest.NewServer(serving)
	t.Cleanup(corpus.Close)
	return markdowncorporahttp.NewMarkdownCorpus(corpus.Listener.Addr().String(), recallDeadline)
}

func TestPageTheCorpusHoldsIsReadWithItsVersionAndTheTimeItWasStored(t *testing.T) {
	storedAt := time.Now().UTC().Truncate(time.Second)
	corpus := openCorpusUnderTest(t, &corpusServing{
		held: &pagemarkdownstore.RecalledPage{
			CanonicalURL: pageAddress,
			Markdown:     heldMarkdown,
			Version:      heldVersion,
			StoredAt:     storedAt,
		},
	})

	held, err := corpus.PageMarkdownAt(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	if held.Markdown != heldMarkdown {
		t.Errorf("markdown = %q, want %q", held.Markdown, heldMarkdown)
	}
	if held.Version != heldVersion {
		t.Errorf("version = %q, want %q", held.Version, heldVersion)
	}
	if !held.StoredAt.Equal(storedAt) {
		t.Errorf("stored at = %v, want %v", held.StoredAt, storedAt)
	}
}

func TestPageTheCorpusDoesNotHoldIsAnsweredAsNotInTheCorpus(t *testing.T) {
	corpus := openCorpusUnderTest(t, &corpusServing{notFound: true})

	_, err := corpus.PageMarkdownAt(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	)
	if !errors.Is(err, pageread.ErrPageNotInCorpus) {
		t.Fatalf("error = %v, want %v", err, pageread.ErrPageNotInCorpus)
	}
}

func TestCorpusThatIsAwayFailsTheRead(t *testing.T) {
	corpus := openCorpusUnderTest(t, &corpusServing{})

	if _, err := corpus.PageMarkdownAt(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, pageAddress),
	); err == nil {
		t.Fatal("the read answered without an error, want the corpus that is away to fail")
	}
}
