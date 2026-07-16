// Package pagefeed binds a representation's derivation to its publication, erasing the
// representation's domain type so the crawler can hold a uniform slice of feeds while
// each still derives and publishes its own R.
package pagefeed

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type bound[R any] struct {
	representation yacycrawlcontract.PageRepresentationKind
	derivation     crawlcapability.RepresentationDerivation[R]
	publication    crawlcapability.PagePublication[R]
}

func Bind[R any](
	representation yacycrawlcontract.PageRepresentationKind,
	derivation crawlcapability.RepresentationDerivation[R],
	publication crawlcapability.PagePublication[R],
) crawlcapability.PageFeed {
	return bound[R]{
		representation: representation,
		derivation:     derivation,
		publication:    publication,
	}
}

func (b bound[R]) Representation() yacycrawlcontract.PageRepresentationKind {
	return b.representation
}

func (b bound[R]) Accepts(format crawlcapability.PageContentFormat) bool {
	return b.derivation.Accepts(format)
}

func (b bound[R]) Derive(
	page crawlcapability.CrawledPage,
	render crawlcapability.RenderContent,
) (crawlcapability.PublishPage, error) {
	derived, err := b.derivation.Derive(page, render)
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		return b.publication.Publish(ctx, derived)
	}, nil
}
