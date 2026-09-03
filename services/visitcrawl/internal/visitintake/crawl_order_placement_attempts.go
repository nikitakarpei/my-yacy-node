package visitintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type CrawlOrderPlacementAttempts struct {
	place    func(ctx context.Context, order yacycrawlcontract.CrawlOrder) error
	observer CrawlOrderPlacementObserver
	timeout  time.Duration
	inFlight chan struct{}
}

func NewCrawlOrderPlacementAttempts(
	place func(ctx context.Context, order yacycrawlcontract.CrawlOrder) error,
	observer CrawlOrderPlacementObserver,
	timeout time.Duration,
	maxInFlight int,
) *CrawlOrderPlacementAttempts {
	return &CrawlOrderPlacementAttempts{
		place:    place,
		observer: observer,
		timeout:  timeout,
		inFlight: make(chan struct{}, maxInFlight),
	}
}

func (p *CrawlOrderPlacementAttempts) Start(order yacycrawlcontract.CrawlOrder) {
	select {
	case p.inFlight <- struct{}{}:
	default:
		p.observer.CrawlOrderPlacementSkippedBecauseSaturated(
			context.Background(), order.OrderID,
		)
		return
	}

	go func() {
		defer func() { <-p.inFlight }()

		ctx, cancel := context.WithTimeout(context.Background(), p.timeout)
		defer cancel()

		if err := p.place(ctx, order); err != nil {
			p.observer.CrawlOrderPlacementFailed(ctx, order.OrderID, err)
			return
		}
		p.observer.CrawlOrderPlaced(ctx, order.OrderID)
	}()
}
