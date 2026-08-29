// Package nats announces on a core NATS subject of its own what became of every scrape
// request the corpus settled, so a caller waiting for one page learns that its markdown is
// stored, or that the page is given up, without polling the corpus. Nothing keeps an
// announcement: a listener that is away when one is announced never learns it.
package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const announcementConfirmationLimit = 10 * time.Second

type ScrapeOutcomeAnnouncements struct {
	connection *nats.Conn
}

func NewScrapeOutcomeAnnouncements(connection *nats.Conn) *ScrapeOutcomeAnnouncements {
	return &ScrapeOutcomeAnnouncements{connection: connection}
}

func (a *ScrapeOutcomeAnnouncements) AnnounceMarkdownStored(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
) error {
	return a.announce(ctx, requestedURL, pagemarkdownstore.MarkdownStored)
}

func (a *ScrapeOutcomeAnnouncements) AnnouncePageGivenUp(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
) error {
	return a.announce(ctx, requestedURL, pagemarkdownstore.PageGivenUp)
}

func (a *ScrapeOutcomeAnnouncements) announce(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	outcome pagemarkdownstore.ScrapeOutcome,
) error {
	notice, err := pagemarkdownstore.MarshalScrapeOutcomeNotice(
		pagemarkdownstore.ScrapeOutcomeNotice{RequestedURL: requestedURL, Outcome: outcome},
	)
	if err != nil {
		return err
	}
	subject := pagemarkdownstore.ScrapeOutcomeSubjectOf(requestedURL)
	if err := a.connection.Publish(subject, notice); err != nil {
		return fmt.Errorf("announce %q for %q: %w", outcome, requestedURL, err)
	}
	confirmationCtx, cancel := context.WithTimeout(ctx, announcementConfirmationLimit)
	defer cancel()
	if err := a.connection.FlushWithContext(confirmationCtx); err != nil {
		return fmt.Errorf("flush the announcement for %q: %w", requestedURL, err)
	}
	return nil
}
