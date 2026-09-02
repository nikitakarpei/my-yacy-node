package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	publicationConfirmationLimit = 10 * time.Second

	msgReceiptSubjectUnknown = "intake receipt carried on no page feed, " +
		"a caller waiting for this page learns nothing until it stops waiting"
	msgReceiptNotCarried = "intake receipt not carried onto the page feed, " +
		"a caller waiting for this page learns nothing until it stops waiting"
)

type ScrapeOutcomeFeed struct {
	connection *nats.Conn
}

func NewScrapeOutcomeFeed(connection *nats.Conn) *ScrapeOutcomeFeed {
	return &ScrapeOutcomeFeed{connection: connection}
}

func (f *ScrapeOutcomeFeed) AnnounceScrapeFailure(
	ctx context.Context,
	failure pagescrapecontract.ScrapeFailure,
) error {
	data, err := pagescrapecontract.MarshalScrapeFailure(failure)
	if err != nil {
		return err
	}
	subject := pagescrapecontract.ScrapeFailureOutcomeSubjectOf(failure.PageURL)
	if err := f.connection.Publish(subject, data); err != nil {
		return fmt.Errorf("announce the failed scrape of %q: %w", failure.PageURL, err)
	}
	return f.confirm(ctx, failure.PageURL.String())
}

func (f *ScrapeOutcomeFeed) CarryIntakeReceipts(ctx context.Context) error {
	subscription, err := f.connection.Subscribe(
		pagescrapecontract.EveryIntakeReceiptSubject,
		func(receipt *nats.Msg) { f.carry(ctx, receipt) },
	)
	if err != nil {
		return fmt.Errorf(
			"subscribe to %q: %w", pagescrapecontract.EveryIntakeReceiptSubject, err,
		)
	}
	<-ctx.Done()
	if err := subscription.Unsubscribe(); err != nil {
		return fmt.Errorf(
			"unsubscribe from %q: %w", pagescrapecontract.EveryIntakeReceiptSubject, err,
		)
	}
	return nil
}

func (f *ScrapeOutcomeFeed) carry(ctx context.Context, receipt *nats.Msg) {
	subject, err := pagescrapecontract.ScrapeOutcomeSubjectFrom(receipt.Subject)
	if err != nil {
		slog.WarnContext(ctx, msgReceiptSubjectUnknown,
			slog.String("receiptSubject", receipt.Subject),
			slog.Any("error", err),
		)
		return
	}
	if err := f.connection.Publish(subject, receipt.Data); err != nil {
		slog.WarnContext(ctx, msgReceiptNotCarried,
			slog.String("receiptSubject", receipt.Subject),
			slog.Any("error", err),
		)
	}
}

func (f *ScrapeOutcomeFeed) confirm(ctx context.Context, pageURL string) error {
	confirmationCtx, cancel := context.WithTimeout(ctx, publicationConfirmationLimit)
	defer cancel()
	if err := f.connection.FlushWithContext(confirmationCtx); err != nil {
		return fmt.Errorf("flush the page feed of %q: %w", pageURL, err)
	}
	return nil
}
