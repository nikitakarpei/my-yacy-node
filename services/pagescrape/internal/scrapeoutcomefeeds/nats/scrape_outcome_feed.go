package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const brokerConfirmationLimit = 10 * time.Second

type ScrapeOutcomeFeed struct {
	connection *nats.Conn
	observer   ScrapeOutcomeFeedObserver
}

func NewScrapeOutcomeFeed(
	connection *nats.Conn,
	observer ScrapeOutcomeFeedObserver,
) *ScrapeOutcomeFeed {
	return &ScrapeOutcomeFeed{connection: connection, observer: observer}
}

func (f *ScrapeOutcomeFeed) AnnounceScrapeFailure(
	ctx context.Context,
	failure pagescrapecontract.ScrapeFailure,
) {
	data, err := pagescrapecontract.MarshalScrapeFailure(failure)
	if err != nil {
		f.observer.ScrapeFailureEncodingFailed(ctx, failure.PageURL, err)
		return
	}
	subject := pagescrapecontract.ScrapeFailureOutcomeSubjectOf(failure.PageURL)
	if err := f.connection.Publish(subject, data); err != nil {
		f.observer.ScrapeFailurePublishingFailed(ctx, failure.PageURL, subject, err)
		return
	}
	f.confirm(ctx, failure.PageURL, subject)
}

func (f *ScrapeOutcomeFeed) confirm(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
) {
	confirmationCtx, cancel := context.WithTimeout(ctx, brokerConfirmationLimit)
	defer cancel()
	if err := f.connection.FlushWithContext(confirmationCtx); err != nil {
		f.observer.ScrapeFailureConfirmationFailed(ctx, pageURL, subject, err)
		return
	}
	f.observer.ScrapeFailureAnnounced(ctx, pageURL, subject)
}

func (f *ScrapeOutcomeFeed) StartCarryingIntakeReceipts(
	ctx context.Context,
) (*IntakeReceiptCarrier, error) {
	subscription, err := f.connection.Subscribe(
		pagescrapecontract.EveryIntakeReceiptSubject,
		func(receipt *nats.Msg) { f.carry(ctx, receipt) },
	)
	if err != nil {
		return nil, fmt.Errorf(
			"subscribe to %q: %w", pagescrapecontract.EveryIntakeReceiptSubject, err,
		)
	}
	if err := f.connection.FlushTimeout(brokerConfirmationLimit); err != nil {
		return nil, fmt.Errorf(
			"confirm the subscription to %q: %w", pagescrapecontract.EveryIntakeReceiptSubject, err,
		)
	}
	return &IntakeReceiptCarrier{subscription: subscription}, nil
}

func (f *ScrapeOutcomeFeed) carry(ctx context.Context, receipt *nats.Msg) {
	subject, err := pagescrapecontract.ScrapeOutcomeSubjectFrom(receipt.Subject)
	if err != nil {
		f.observer.IntakeReceiptSubjectUnreadable(ctx, receipt.Subject, err)
		return
	}
	if err := f.connection.Publish(subject, receipt.Data); err != nil {
		f.observer.IntakeReceiptNotCarried(ctx, receipt.Subject, subject, err)
		return
	}
	f.observer.IntakeReceiptCarried(ctx, receipt.Subject, subject)
}

type IntakeReceiptCarrier struct {
	subscription *nats.Subscription
}

func (c *IntakeReceiptCarrier) Run(ctx context.Context) error {
	<-ctx.Done()
	if err := c.subscription.Unsubscribe(); err != nil {
		return fmt.Errorf(
			"unsubscribe from %q: %w", pagescrapecontract.EveryIntakeReceiptSubject, err,
		)
	}
	return nil
}
