package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const publicationConfirmationLimit = 10 * time.Second

type IntakeReceipts struct {
	connection *nats.Conn
	corpus     pagescrapecontract.CorpusName
}

func NewIntakeReceipts(
	connection *nats.Conn,
	corpus pagescrapecontract.CorpusName,
) *IntakeReceipts {
	return &IntakeReceipts{connection: connection, corpus: corpus}
}

func (r *IntakeReceipts) ReportKeptPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) error {
	data, err := pagescrapecontract.MarshalKeptPage(pagescrapecontract.KeptPage{
		PageURL: pageURL,
		Corpus:  r.corpus,
	})
	if err != nil {
		return err
	}
	return r.send(ctx, pagescrapecontract.KeptPageSubjectOf(pageURL), data, pageURL)
}

func (r *IntakeReceipts) ReportRejectedPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) error {
	data, err := pagescrapecontract.MarshalRejectedPage(pagescrapecontract.RejectedPage{
		PageURL: pageURL,
		Corpus:  r.corpus,
	})
	if err != nil {
		return err
	}
	return r.send(ctx, pagescrapecontract.RejectedPageSubjectOf(pageURL), data, pageURL)
}

func (r *IntakeReceipts) send(
	ctx context.Context,
	subject string,
	data []byte,
	pageURL canonicalurl.CanonicalURL,
) error {
	if err := r.connection.Publish(subject, data); err != nil {
		return fmt.Errorf("send the intake receipt for %q: %w", pageURL, err)
	}
	confirmationCtx, cancel := context.WithTimeout(ctx, publicationConfirmationLimit)
	defer cancel()
	if err := r.connection.FlushWithContext(confirmationCtx); err != nil {
		return fmt.Errorf("confirm the intake receipt for %q: %w", pageURL, err)
	}
	return nil
}
