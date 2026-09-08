package http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	markdownrecallreceivershttp "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecallreceivers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/markdowncorpusclienttest"
)

const (
	recallTimeLimit = 10 * time.Second
	recalledURL     = "https://example.com/"
)

var errCorpusUnreachable = errors.New("corpus unreachable")

var storedAt = time.Date(2026, time.August, 25, 10, 30, 0, 0, time.UTC)

var heldMarkdown = markdownrecall.StoredMarkdown{
	Markdown: []byte("# Hi"),
	StoredAt: storedAt,
	Version:  "SHA-256=jbHfCoRuBAP7Kzb9EGMSJnEcVYYnvKvJcYCA1LMKGVw=",
}

type recalledPages struct {
	byRequestedURL map[canonicalurl.CanonicalURL]markdownrecall.PageUnderRequestedURL
	failure        error
}

func (r recalledPages) PageOf(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
) (markdownrecall.PageUnderRequestedURL, bool, error) {
	if r.failure != nil {
		return markdownrecall.PageUnderRequestedURL{}, false, r.failure
	}
	page, held := r.byRequestedURL[requestedURL]
	if !held {
		return markdownrecall.PageUnderRequestedURL{}, false, nil
	}
	return page, true, nil
}

type receiverUnderTest struct {
	t      *testing.T
	client markdowncorpusclienttest.MarkdownCorpusClient
}

func markdownRecallReceiverUnderTest(
	t *testing.T,
	recall markdownrecallreceivershttp.PageMarkdownRecall,
) receiverUnderTest {
	t.Helper()
	receiver := httptest.NewServer(markdownrecallreceivershttp.NewMux(recall))
	t.Cleanup(receiver.Close)
	return receiverUnderTest{
		t:      t,
		client: markdowncorpusclienttest.New(t, receiver.Listener.Addr().String()),
	}
}

func (r receiverUnderTest) RecallPage(url string) (pagemarkdownstore.RecalledPage, int) {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), recallTimeLimit)
	defer cancel()
	return r.client.RecallPage(ctx, url)
}

func TestRecallPageAnswersWithTheMarkdownTheCorpusHolds(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{
		byRequestedURL: map[canonicalurl.CanonicalURL]markdownrecall.PageUnderRequestedURL{
			canonicalurltest.CanonicalURLOf(t, recalledURL): {
				MarkdownURL:    canonicalurltest.CanonicalURLOf(t, recalledURL),
				StoredMarkdown: heldMarkdown,
			},
		},
	})

	recalled, statusCode := receiver.RecallPage(recalledURL)

	if statusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", statusCode, http.StatusOK)
	}
	if recalled.CanonicalURL != recalledURL {
		t.Errorf("canonicalUrl = %q, want %q", recalled.CanonicalURL, recalledURL)
	}
	if recalled.Markdown != "# Hi" {
		t.Errorf("markdown = %q, want %q", recalled.Markdown, "# Hi")
	}
	if !recalled.StoredAt.Equal(storedAt) {
		t.Errorf("storedAt = %v, want %v", recalled.StoredAt, storedAt)
	}
	if recalled.Version != heldMarkdown.Version {
		t.Errorf("version = %q, want %q", recalled.Version, heldMarkdown.Version)
	}
}

func TestRecallPageAnswersWithTheURLTheMarkdownIsOf(t *testing.T) {
	const redirectedFrom = "http://example.com/"
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{
		byRequestedURL: map[canonicalurl.CanonicalURL]markdownrecall.PageUnderRequestedURL{
			canonicalurltest.CanonicalURLOf(t, redirectedFrom): {
				MarkdownURL:    canonicalurltest.CanonicalURLOf(t, recalledURL),
				StoredMarkdown: heldMarkdown,
			},
		},
	})

	recalled, _ := receiver.RecallPage(redirectedFrom)

	if recalled.CanonicalURL != recalledURL {
		t.Errorf("canonicalUrl = %q, want %q", recalled.CanonicalURL, recalledURL)
	}
}

func TestRecallPageReportsNotFoundForAURLTheCorpusDoesNotHold(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{})

	_, statusCode := receiver.RecallPage(recalledURL)

	if statusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", statusCode, http.StatusNotFound)
	}
}

func TestRecallPageRefusesARequestWhoseURLIsNotCanonicalizable(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{})

	_, statusCode := receiver.RecallPage("")

	if statusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", statusCode, http.StatusBadRequest)
	}
}

func TestRecallPageReportsACorpusFailureAsInternal(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{failure: errCorpusUnreachable})

	_, statusCode := receiver.RecallPage(recalledURL)

	if statusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", statusCode, http.StatusInternalServerError)
	}
}
