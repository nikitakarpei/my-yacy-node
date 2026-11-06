package jetstream

import (
	"context"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
)

type PageRWIPublisher struct {
	pageRWIs jetstream.JetStream
	observer PageRWIPublicationObserver
}

func NewPageRWIPublisher(
	pageRWIs jetstream.JetStream,
	observer PageRWIPublicationObserver,
) *PageRWIPublisher {
	return &PageRWIPublisher{pageRWIs: pageRWIs, observer: observer}
}

func (p *PageRWIPublisher) Publish(ctx context.Context, pageRWI pagerwicontract.PageRWI) bool {
	data, err := pagerwicontract.MarshalPageRWI(pageRWI)
	if err != nil {
		p.observer.PageRWIEncodingFailed(ctx, pageRWI.PageURL, err)
		return false
	}
	if _, err := p.pageRWIs.Publish(ctx, pagerwicontract.PageRWISubject, data); err != nil {
		p.observer.PageRWIPublishingFailed(ctx, pageRWI.PageURL, err)
		return false
	}
	p.observer.PageRWIPublished(ctx, pageRWI.PageURL)
	return true
}
