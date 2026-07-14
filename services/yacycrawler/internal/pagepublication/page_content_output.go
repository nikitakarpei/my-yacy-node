package pagepublication

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const msgContentNotDerivable = "page content representation not derivable from extracted body"

type PageContentOutput struct {
	publisher  jetstream.JetStream
	subject    string
	derivation crawlcapability.ContentDerivation
}

func NewPageContentOutput(
	publisher jetstream.JetStream,
	subject string,
	derivation crawlcapability.ContentDerivation,
) PageContentOutput {
	return PageContentOutput{publisher: publisher, subject: subject, derivation: derivation}
}

func (o PageContentOutput) Name() string {
	return string(pageRepresentationOf(o.derivation.Format()))
}

func (o PageContentOutput) Publish(ctx context.Context, page crawlcapability.ExtractedPage) error {
	body, err := o.derivation.Derive(page.Body, page.Format)
	if errors.Is(err, crawlcapability.ErrUnsupportedSourceFormat) {
		slog.WarnContext(ctx, msgContentNotDerivable,
			slog.String("url", page.CanonicalURL),
			slog.String("sourceFormat", string(page.Format)),
			slog.String("representation", o.Name()))
		return nil
	}
	if err != nil {
		return fmt.Errorf("derive page content: %w", err)
	}

	payload, err := yacycrawlcontract.MarshalPageContentRepresentation(
		yacycrawlcontract.PageContentRepresentation{
			CanonicalURL: page.CanonicalURL,
			Title:        page.Title,
			CrawledAt:    page.FetchedAt,
			Language:     page.Language,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("marshal page content representation: %w", err)
	}
	if _, err := o.publisher.Publish(ctx, o.subject, payload); err != nil {
		return classifyPublishError(err)
	}
	return nil
}
