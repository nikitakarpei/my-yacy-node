// Package background places a crawl order out of the caller's turn: Place
// takes the order and returns at once, then a crawl order placer carries it to
// its destination under a time bound. Only a set number of placements are in
// flight at the same time; Place refuses an order it has no room for. The
// observer reports whether the order was accepted or refused; how an accepted
// order then fared is reported by the placer that carries it.
package background

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type BackgroundCrawlOrderPlacer struct {
	placer   CrawlOrderPlacer
	observer BackgroundCrawlOrderPlacementObserver
	timeout  time.Duration
	inFlight chan struct{}
}

type CrawlOrderPlacer interface {
	Place(ctx context.Context, order yacycrawlcontract.CrawlOrder)
}

func New(
	placer CrawlOrderPlacer,
	observer BackgroundCrawlOrderPlacementObserver,
	timeout time.Duration,
	maxInFlight int,
) *BackgroundCrawlOrderPlacer {
	return &BackgroundCrawlOrderPlacer{
		placer:   placer,
		observer: observer,
		timeout:  timeout,
		inFlight: make(chan struct{}, maxInFlight),
	}
}

func (p *BackgroundCrawlOrderPlacer) Place(
	ctx context.Context,
	order yacycrawlcontract.CrawlOrder,
) {
	select {
	case p.inFlight <- struct{}{}:
	default:
		p.observer.CrawlOrderPlacementRefused(ctx, order.OrderID)
		return
	}
	p.observer.CrawlOrderPlacementAccepted(ctx, order.OrderID)
	go p.placeWithinTimeout(context.WithoutCancel(ctx), order)
}

func (p *BackgroundCrawlOrderPlacer) placeWithinTimeout(
	ctx context.Context,
	order yacycrawlcontract.CrawlOrder,
) {
	defer func() { <-p.inFlight }()

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	p.placer.Place(ctx, order)
}
