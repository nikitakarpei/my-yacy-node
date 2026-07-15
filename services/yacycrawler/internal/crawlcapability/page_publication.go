package crawlcapability

import "context"

type PagePublication[R any] interface {
	Publish(ctx context.Context, representation R) error
}

// PageRepresentationOutput erases a representation's domain type so the crawler can
// hold a uniform slice of outputs while each still derives and publishes its own R.
type PageRepresentationOutput struct {
	name    string
	accepts func(PageContentFormat) bool
	prepare func(CrawledPage, *RenderedContent) (func(context.Context) error, error)
}

func BindRepresentation[R any](
	derivation RepresentationDerivation[R],
	publication PagePublication[R],
) PageRepresentationOutput {
	return PageRepresentationOutput{
		name:    derivation.Name(),
		accepts: derivation.Accepts,
		prepare: func(page CrawledPage, rendered *RenderedContent) (func(context.Context) error, error) {
			representation, err := derivation.Derive(page, rendered)
			if err != nil {
				return nil, err
			}
			return func(ctx context.Context) error {
				return publication.Publish(ctx, representation)
			}, nil
		},
	}
}

func (o PageRepresentationOutput) Name() string {
	return o.name
}

func (o PageRepresentationOutput) Accepts(format PageContentFormat) bool {
	return o.accepts(format)
}

func (o PageRepresentationOutput) Prepare(
	page CrawledPage,
	rendered *RenderedContent,
) (func(context.Context) error, error) {
	return o.prepare(page, rendered)
}
