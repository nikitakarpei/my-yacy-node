package nats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	scrapeoutcomeannouncementsnats "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/scrapeoutcomeannouncements/nats"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const (
	requestedURL      = "https://example.com/"
	otherRequestedURL = "https://example.com/other"
	announcementWait  = 5 * time.Second
)

func TestAnnounceMarkdownStoredCarriesTheStoredOutcomeOfTheRequestedURL(t *testing.T) {
	announcements, listener := announcementsUnderTest(t)
	pageURL := canonicalurltest.CanonicalURLOf(t, requestedURL)

	if err := announcements.AnnounceMarkdownStored(context.Background(), pageURL); err != nil {
		t.Fatalf("announce markdown stored: %v", err)
	}

	notice := heardNotice(t, listener)
	if notice.RequestedURL != pageURL {
		t.Errorf("announced url = %q, want %q", notice.RequestedURL, pageURL)
	}
	if notice.Outcome != pagemarkdownstore.MarkdownStored {
		t.Errorf(
			"announced outcome = %q, want %q",
			notice.Outcome,
			pagemarkdownstore.MarkdownStored,
		)
	}
}

func TestAnnouncePageGivenUpCarriesTheGivenUpOutcomeOfTheRequestedURL(t *testing.T) {
	announcements, listener := announcementsUnderTest(t)
	pageURL := canonicalurltest.CanonicalURLOf(t, requestedURL)

	if err := announcements.AnnouncePageGivenUp(context.Background(), pageURL); err != nil {
		t.Fatalf("announce page given up: %v", err)
	}

	notice := heardNotice(t, listener)
	if notice.Outcome != pagemarkdownstore.PageGivenUp {
		t.Errorf("announced outcome = %q, want %q", notice.Outcome, pagemarkdownstore.PageGivenUp)
	}
}

func TestAnnouncementsOfOnePageReachNoListenerOfAnother(t *testing.T) {
	announcements, listener := announcementsUnderTest(t)

	err := announcements.AnnounceMarkdownStored(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, otherRequestedURL),
	)
	if err != nil {
		t.Fatalf("announce markdown stored: %v", err)
	}

	if _, err := listener.NextMsg(time.Second); err == nil {
		t.Error("a listener of one page heard the announcement of another")
	}
}

func TestAnnouncementsStopWhenTheCallerStopsWaiting(t *testing.T) {
	announcements, _ := announcementsUnderTest(t)
	ctx, stopWaiting := context.WithCancel(context.Background())
	stopWaiting()

	err := announcements.AnnounceMarkdownStored(
		ctx,
		canonicalurltest.CanonicalURLOf(t, requestedURL),
	)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("announce markdown stored = %v, want the cancellation", err)
	}
}

func announcementsUnderTest(
	t *testing.T,
) (*scrapeoutcomeannouncementsnats.ScrapeOutcomeAnnouncements, *nats.Subscription) {
	t.Helper()
	url := natstestserver.Start(t)
	listener := listenerOf(t, url, canonicalurltest.CanonicalURLOf(t, requestedURL))
	return scrapeoutcomeannouncementsnats.NewScrapeOutcomeAnnouncements(
		natstestserver.Connect(t, url),
	), listener
}

func listenerOf(t *testing.T, url string, pageURL canonicalurl.CanonicalURL) *nats.Subscription {
	t.Helper()
	connection := natstestserver.Connect(t, url)
	listener, err := connection.SubscribeSync(pagemarkdownstore.ScrapeOutcomeSubjectOf(pageURL))
	if err != nil {
		t.Fatalf("listen for scrape outcomes: %v", err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("flush the listener: %v", err)
	}
	return listener
}

func heardNotice(t *testing.T, listener *nats.Subscription) pagemarkdownstore.ScrapeOutcomeNotice {
	t.Helper()
	message, err := listener.NextMsg(announcementWait)
	if err != nil {
		t.Fatalf("wait for the announcement: %v", err)
	}
	notice, err := pagemarkdownstore.UnmarshalScrapeOutcomeNotice(message.Data)
	if err != nil {
		t.Fatalf("unmarshal the announcement: %v", err)
	}
	return notice
}
