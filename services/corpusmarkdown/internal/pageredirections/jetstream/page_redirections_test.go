package jetstream_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	pageredirectionsjetstream "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageredirections/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
)

const (
	requestedURL = "http://example.com/page"
	settledURL   = "https://example.com/page"
)

func openedRedirections(t *testing.T) *pageredirectionsjetstream.PageRedirections {
	t.Helper()
	redirections, err := pageredirectionsjetstream.OpenPageRedirections(
		context.Background(),
		natstestserver.ConnectJetStream(t, natstestserver.Start(t)),
	)
	if err != nil {
		t.Fatalf("open page redirections: %v", err)
	}
	return redirections
}

func TestRedirectionOfYieldsTheURLTheRequestedURLLedTo(t *testing.T) {
	redirections := openedRedirections(t)
	if err := redirections.Record(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, requestedURL),
		canonicalurltest.CanonicalURLOf(t, settledURL),
	); err != nil {
		t.Fatalf("record: %v", err)
	}

	markdownURL, redirected, err := redirections.RedirectionOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, requestedURL),
	)
	if err != nil {
		t.Fatalf("redirection of %q: %v", requestedURL, err)
	}
	if !redirected {
		t.Fatalf("no redirection held for %q", requestedURL)
	}
	if markdownURL.String() != settledURL {
		t.Errorf("markdownURL = %q, want %q", markdownURL, settledURL)
	}
}

func TestRecordingAgainReplacesTheURLARequestedURLLeadsTo(t *testing.T) {
	const movedAgainURL = "https://www.example.com/page"
	redirections := openedRedirections(t)
	for _, settled := range []string{settledURL, movedAgainURL} {
		if err := redirections.Record(
			context.Background(),
			canonicalurltest.CanonicalURLOf(t, requestedURL),
			canonicalurltest.CanonicalURLOf(t, settled),
		); err != nil {
			t.Fatalf("record %q: %v", settled, err)
		}
	}

	markdownURL, _, err := redirections.RedirectionOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, requestedURL),
	)
	if err != nil {
		t.Fatalf("redirection of %q: %v", requestedURL, err)
	}
	if markdownURL.String() != movedAgainURL {
		t.Errorf("markdownURL = %q, want %q", markdownURL, movedAgainURL)
	}
}

func TestRedirectionOfReportsAURLThatNeverRedirected(t *testing.T) {
	_, redirected, err := openedRedirections(t).RedirectionOf(
		context.Background(), canonicalurltest.CanonicalURLOf(t, requestedURL),
	)
	if err != nil {
		t.Fatalf("redirection of %q: %v", requestedURL, err)
	}
	if redirected {
		t.Fatalf("a redirection was held for %q, which never redirected", requestedURL)
	}
}

func TestOpenPageRedirectionsFailsWhenTheBucketCannotBeCreated(t *testing.T) {
	pageMarkdownJetStream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, err := pageredirectionsjetstream.OpenPageRedirections(
		abandoned, pageMarkdownJetStream,
	); err == nil {
		t.Fatal("expected error when the page redirection bucket cannot be created")
	}
}

func TestRecordFailsWhenTheBucketCannotBeWritten(t *testing.T) {
	redirections := openedRedirections(t)
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if err := redirections.Record(
		abandoned,
		canonicalurltest.CanonicalURLOf(t, requestedURL),
		canonicalurltest.CanonicalURLOf(t, settledURL),
	); err == nil {
		t.Fatal("expected error when the redirection cannot be written")
	}
}

func TestRedirectionOfFailsWhenTheBucketCannotBeRead(t *testing.T) {
	redirections := openedRedirections(t)
	abandoned, abandon := context.WithCancel(context.Background())
	abandon()

	if _, _, err := redirections.RedirectionOf(
		abandoned, canonicalurltest.CanonicalURLOf(t, requestedURL),
	); err == nil {
		t.Fatal("expected error when the redirection cannot be read")
	}
}
