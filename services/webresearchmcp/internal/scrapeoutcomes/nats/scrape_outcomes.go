// Package nats listens on the feed the scrape service carries the end of one scrape on, so a
// caller waiting for one page learns what became of it without polling a corpus. Nothing keeps
// a message, so a listener must be open before the scrape is asked for: opening one is the only
// way to reach the wait. A message this feed cannot read goes to the observer, and the listener
// keeps waiting.
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
	observer   ScrapeOutcomeObserver
}

func OpenScrapeOutcomes(
	natsURL string,
	observer ScrapeOutcomeObserver,
) (*ScrapeOutcomes, error) {
	connection, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("connect to nats at %s: %w", natsURL, err)
	}
	return &ScrapeOutcomes{connection: connection, observer: observer}, nil
}

func (o *ScrapeOutcomes) ListenerFor(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) (pageread.ScrapeOutcomeListener, error) {
	subject := pagescrapecontract.ScrapeOutcomeSubjectsOf(pageURL)
	subscription, err := o.connection.SubscribeSync(subject)
	if err != nil {
		o.observer.ScrapeOutcomeSubscriptionFailed(ctx, pageURL, err)
		return nil, fmt.Errorf("listen for the scrape outcome of %q: %w", pageURL, err)
	}
	if err := o.connection.Flush(); err != nil {
		_ = subscription.Unsubscribe()
		o.observer.ScrapeOutcomeListenerConfirmationFailed(ctx, pageURL, err)
		return nil, fmt.Errorf("confirm the listener for %q: %w", pageURL, err)
	}
	return &scrapeOutcomeListener{
		subscription: subscription,
		pageURL:      pageURL,
		observer:     o.observer,
	}, nil
}

func (o *ScrapeOutcomes) Close() {
	o.connection.Close()
}

type scrapeOutcomeListener struct {
	subscription *nats.Subscription
	pageURL      canonicalurl.CanonicalURL
	observer     ScrapeOutcomeObserver
}

func (l *scrapeOutcomeListener) AwaitedFetchOutcome(
	ctx context.Context,
) pageread.FetchOutcome {
	for {
		message, err := l.subscription.NextMsgWithContext(ctx)
		if err != nil {
			return pageread.FetchUnfinished
		}
		outcome, err := pagescrapecontract.ScrapeOutcomeOn(message.Subject)
		if err != nil {
			l.observer.ScrapeOutcomeMessageMalformed(ctx, l.pageURL, err)
			continue
		}
		if outcome == pagescrapecontract.ScrapeFailed {
			return pageread.PageNotReadable
		}
		receiptCorpus, err := intakeReceiptCorpusFrom(message.Data, outcome)
		if err != nil {
			l.observer.ScrapeOutcomeMessageMalformed(ctx, l.pageURL, err)
			continue
		}
		if receiptCorpus == pagescrapecontract.CorpusMarkdown {
			return fetchOutcomeOf(outcome)
		}
	}
}

func intakeReceiptCorpusFrom(
	data []byte,
	outcome pagescrapecontract.ScrapeOutcome,
) (pagescrapecontract.CorpusName, error) {
	if outcome == pagescrapecontract.PageKept {
		keptPage, err := pagescrapecontract.UnmarshalKeptPage(data)
		if err != nil {
			return "", err
		}
		return keptPage.Corpus, nil
	}
	rejectedPage, err := pagescrapecontract.UnmarshalRejectedPage(data)
	if err != nil {
		return "", err
	}
	return rejectedPage.Corpus, nil
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
