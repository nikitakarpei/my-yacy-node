// Package nats listens on the feed the scrape service carries the end of one scrape on, so a
// caller waiting for one page learns what became of it without polling a corpus. Nothing keeps
// a message, so a listener must be open before the scrape is asked for: opening one is the only
// way to reach the wait.
package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

type ScrapeOutcomes struct {
	connection *nats.Conn
}

func OpenScrapeOutcomes(natsURL string) (*ScrapeOutcomes, error) {
	connection, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("connect to nats at %s: %w", natsURL, err)
	}
	return &ScrapeOutcomes{connection: connection}, nil
}

func (o *ScrapeOutcomes) ListenerFor(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) (pageread.ScrapeOutcomeListener, error) {
	subject := pagescrapecontract.ScrapeOutcomeSubjectsOf(pageURL)
	subscription, err := o.connection.SubscribeSync(subject)
	if err != nil {
		return nil, fmt.Errorf("listen for the scrape outcome of %q: %w", pageURL, err)
	}
	if err := o.connection.Flush(); err != nil {
		_ = subscription.Unsubscribe()
		return nil, fmt.Errorf("confirm the listener for %q: %w", pageURL, err)
	}
	return &scrapeOutcomeListener{subscription: subscription, pageURL: pageURL}, nil
}

func (o *ScrapeOutcomes) Close() {
	o.connection.Close()
}

type scrapeOutcomeListener struct {
	subscription *nats.Subscription
	pageURL      canonicalurl.CanonicalURL
}

func (l *scrapeOutcomeListener) AwaitedFetchOutcome(
	ctx context.Context,
) (pageread.FetchOutcome, error) {
	message, err := l.subscription.NextMsgWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("wait for the scrape outcome of %q: %w", l.pageURL, err)
	}
	outcome, err := pagescrapecontract.ScrapeOutcomeOn(message.Subject)
	if err != nil {
		return "", err
	}
	return fetchOutcomeOf(outcome), nil
}

func fetchOutcomeOf(outcome pagescrapecontract.ScrapeOutcome) pageread.FetchOutcome {
	if outcome == pagescrapecontract.PageKept {
		return pageread.PageFetched
	}
	return pageread.PageNotReadable
}

func (l *scrapeOutcomeListener) Close() {
	_ = l.subscription.Unsubscribe()
}
