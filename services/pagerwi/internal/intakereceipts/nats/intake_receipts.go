package nats

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const publicationConfirmationLimit = 10 * time.Second

type IntakeReceipts struct {
	connection *nats.Conn
	corpus     pagescrapecontract.CorpusName
	observer   IntakeReceiptPublicationObserver
}

func NewIntakeReceipts(
	connection *nats.Conn,
	corpus pagescrapecontract.CorpusName,
	observer IntakeReceiptPublicationObserver,
) *IntakeReceipts {
	return &IntakeReceipts{connection: connection, corpus: corpus, observer: observer}
}

func (r *IntakeReceipts) ReportKeptPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	data, err := pagescrapecontract.MarshalKeptPage(pagescrapecontract.KeptPage{
		PageURL: pageURL,
		Corpus:  r.corpus,
	})
	if err != nil {
		r.observer.IntakeReceiptEncodingFailed(ctx, pageURL, err)
		return
	}
	r.send(ctx, pagescrapecontract.KeptPageSubjectOf(pageURL), data, pageURL)
}

func (r *IntakeReceipts) ReportRejectedPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	data, err := pagescrapecontract.MarshalRejectedPage(pagescrapecontract.RejectedPage{
		PageURL: pageURL,
		Corpus:  r.corpus,
	})
	if err != nil {
		r.observer.IntakeReceiptEncodingFailed(ctx, pageURL, err)
		return
	}
	r.send(ctx, pagescrapecontract.RejectedPageSubjectOf(pageURL), data, pageURL)
}

func (r *IntakeReceipts) send(
	ctx context.Context,
	subject string,
	data []byte,
	pageURL canonicalurl.CanonicalURL,
) {
	if err := r.connection.Publish(subject, data); err != nil {
		r.observer.IntakeReceiptPublishingFailed(ctx, pageURL, subject, err)
		return
	}
	confirmationCtx, cancel := context.WithTimeout(ctx, publicationConfirmationLimit)
	defer cancel()
	if err := r.connection.FlushWithContext(confirmationCtx); err != nil {
		r.observer.IntakeReceiptConfirmationFailed(ctx, pageURL, subject, err)
		return
	}
	r.observer.IntakeReceiptSent(ctx, pageURL, subject)
}
