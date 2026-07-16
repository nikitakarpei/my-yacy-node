package crawlcapability

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type PagePublication[R any] interface {
	Publish(ctx context.Context, representation R) error
}

// PageRepresentationOutput erases a representation's domain type so the crawler can
// hold a uniform slice of outputs while each still derives and publishes its own R.
type PageRepresentationOutput struct {
	representation yacycrawlcontract.PageRepresentation
	accepts        func(PageContentFormat) bool
	prepare        func(CrawledPage, RenderContent) (func(context.Context) error, error)
}

func BindRepresentation[R any](
	representation yacycrawlcontract.PageRepresentation,
	derivation RepresentationDerivation[R],
	publication PagePublication[R],
) PageRepresentationOutput {
	return PageRepresentationOutput{
		representation: representation,
		accepts:        derivation.Accepts,
		prepare: func(page CrawledPage, render RenderContent) (func(context.Context) error, error) {
			derived, err := derivation.Derive(page, render)
			if err != nil {
				return nil, err
			}
			return func(ctx context.Context) error {
				return publication.Publish(ctx, derived)
			}, nil
		},
	}
}

func (o PageRepresentationOutput) Representation() yacycrawlcontract.PageRepresentation {
	return o.representation
}

func (o PageRepresentationOutput) Accepts(format PageContentFormat) bool {
	return o.accepts(format)
}

func (o PageRepresentationOutput) Prepare(
	page CrawledPage,
	render RenderContent,
) (func(context.Context) error, error) {
	return o.prepare(page, render)
}
