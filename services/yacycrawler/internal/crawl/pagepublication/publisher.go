// Package pagepublication derives each representation's content format from a page and sends it.
package pagepublication

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

const msgPublicationBackpressure = "publication backpressure, awaiting retry"

const msgRepresentationUnderivable = "page format derives no content for this representation, representation skipped"

type Publisher struct {
	graph           contentformatgraph.FormatDerivations
	representations []PageRepresentation
	observer        PublicationProgress
	clock           clock.Clock
	retry           retrydelay.Bounds
}

func New(
	graph contentformatgraph.FormatDerivations,
	representations []PageRepresentation,
	observer PublicationProgress,
	clock clock.Clock,
	retry retrydelay.Bounds,
) *Publisher {
	return &Publisher{
		graph:           graph,
		representations: representations,
		observer:        observer,
		clock:           clock,
		retry:           retry,
	}
}

func (p *Publisher) Publish(ctx context.Context, page Page) error {
	derived, err := p.derivableContentsFor(ctx, page)
	if err != nil {
		return err
	}
	if len(derived) == 0 {
		return fmt.Errorf(
			"page %s: format %s derives no enabled representation",
			page.CanonicalURL.String(), page.Format,
		)
	}
	for _, content := range derived {
		if err := p.publishContent(ctx, page, content); err != nil {
			return err
		}
	}
	return nil
}

type representationContent struct {
	representation PageRepresentation
	content        []byte
}

func (p *Publisher) derivableContentsFor(
	ctx context.Context,
	page Page,
) ([]representationContent, error) {
	resolver := p.graph.ForPage(page.CanonicalURL.String(), page.Format, page.Body)
	derived := make([]representationContent, 0, len(p.representations))
	for _, representation := range p.representations {
		content, resolved, err := resolver.Resolve(representation.ContentFormat())
		if err != nil {
			return nil, err
		}
		if !resolved {
			slog.WarnContext(ctx, msgRepresentationUnderivable,
				slog.String("representation", string(representation.Kind())),
				slog.String("url", page.CanonicalURL.String()),
				slog.String("format", string(page.Format)),
			)
			p.observer.RepresentationUnderivable(representation.Kind())
			continue
		}
		derived = append(derived, representationContent{
			representation: representation,
			content:        content,
		})
	}
	return derived, nil
}

func (p *Publisher) publishContent(
	ctx context.Context,
	page Page,
	derived representationContent,
) error {
	messages, err := derived.representation.Frame(page, derived.content)
	if err != nil {
		return fmt.Errorf("frame %s: %w", derived.representation.Kind(), err)
	}
	if err := p.send(ctx, page, derived.representation, messages); err != nil {
		return err
	}
	p.observer.PagePublished(derived.representation.Kind())
	return nil
}

func (p *Publisher) send(
	ctx context.Context,
	page Page,
	representation PageRepresentation,
	messages [][]byte,
) error {
	waits := 0
	for {
		err := representation.Publish(ctx, messages)
		if err == nil {
			return nil
		}
		var retryable TransientPublicationError
		if !errors.As(err, &retryable) {
			return fmt.Errorf("publish to %s: %w", representation.Kind(), err)
		}
		p.observer.PublicationWaited()
		waits++
		slog.WarnContext(ctx, msgPublicationBackpressure,
			slog.String("representation", string(representation.Kind())),
			slog.String("url", page.CanonicalURL.String()),
			slog.Int("waits", waits),
			slog.Any("error", err),
		)
		if sleepErr := p.clock.Sleep(ctx, p.retry.Delay(waits)); sleepErr != nil {
			return fmt.Errorf("await publication retry: %w", sleepErr)
		}
	}
}
