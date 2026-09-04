package crawlorderbroker

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PublishingCrawlOrderPlacer struct {
	js       jetstream.JetStream
	subject  string
	observer CrawlOrderPublicationObserver
}

func newPublishingCrawlOrderPlacer(
	js jetstream.JetStream,
	subject string,
	observer CrawlOrderPublicationObserver,
) *PublishingCrawlOrderPlacer {
	return &PublishingCrawlOrderPlacer{js: js, subject: subject, observer: observer}
}

func (p *PublishingCrawlOrderPlacer) Place(
	ctx context.Context,
	order yacycrawlcontract.CrawlOrder,
) {
	data, err := yacycrawlcontract.MarshalCrawlOrder(order)
	if err != nil {
		p.observer.CrawlOrderEncodingFailed(ctx, order.OrderID, err)
		return
	}
	if _, err := p.js.Publish(ctx, p.subject, data); err != nil {
		p.observer.CrawlOrderPublishingFailed(ctx, order.OrderID, p.subject, err)
		return
	}
	p.observer.CrawlOrderPublished(ctx, order.OrderID, p.subject)
}
