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

const msgRepresentationUnresolvable = "enabled representation could not derive its content format, page not published"

var ErrRepresentationUnresolvable = errors.New("representation content format unresolvable")

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

func (p *Publisher) Publish(ctx context.Context, page Page) (bool, error) {
	resolver := p.graph.ForPage(page.CanonicalURL, page.Format, page.Body)
	contents := make([][]byte, len(p.representations))
	for i, representation := range p.representations {
		content, resolved, err := resolver.Resolve(representation.ContentFormat())
		if err != nil {
			return false, err
		}
		if !resolved {
			slog.ErrorContext(ctx, msgRepresentationUnresolvable,
				slog.String("representation", string(representation.Kind())),
				slog.String("url", page.CanonicalURL),
				slog.String("format", string(page.Format)),
			)
			return false, fmt.Errorf(
				"%s: %w", representation.Kind(), ErrRepresentationUnresolvable,
			)
		}
		contents[i] = content
	}
	for i, representation := range p.representations {
		messages, err := representation.Frame(page, contents[i])
		if err != nil {
			return false, fmt.Errorf("frame %s: %w", representation.Kind(), err)
		}
		if err := p.send(ctx, page, representation, messages); err != nil {
			return false, err
		}
		p.observer.PagePublished(representation.Kind())
	}
	return true, nil
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
			slog.String("url", page.CanonicalURL),
			slog.Int("waits", waits),
			slog.Any("error", err),
		)
		if sleepErr := p.clock.Sleep(ctx, p.retry.Delay(waits)); sleepErr != nil {
			return fmt.Errorf("await publication retry: %w", sleepErr)
		}
	}
}
