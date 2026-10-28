// Package pagereading reads what a fetched page states: the refusals of its headers and
// of its body, and the URLs it links to when it does not refuse link discovery. The first
// page format that knows the body reads it; a body no format knows links nowhere. An
// unparsed content type and a refused link discovery go to the observer.
package pagereading

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/httpheader"
)

type PageFormat interface {
	ReadingOfBody(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		contentType ContentType,
		body []byte,
	) (BodyReading, bool, error)
}

type Reading struct {
	Refusals       robotsmeta.Refusals
	DiscoveredURLs []canonicalurl.CanonicalURL
}

type PageReader struct {
	pageFormats []PageFormat
	observer    PageReadingObserver
}

func NewPageReader(pageFormats []PageFormat, observer PageReadingObserver) *PageReader {
	return &PageReader{pageFormats: pageFormats, observer: observer}
}

func (reader *PageReader) ReadingOfPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	page pagefetch.FetchedPage,
) (Reading, error) {
	headerRefusals := httpheader.RefusalsOf(page.RobotsTagValues)
	bodyReading, err := reader.readingOfBody(
		ctx, pageURL, reader.contentTypeOf(ctx, page.ContentType), page.Body,
	)
	if err != nil {
		return Reading{}, err
	}
	refusals := headerRefusals.With(bodyReading.Refusals)
	return Reading{
		Refusals: refusals,
		DiscoveredURLs: reader.discoveredURLsFrom(
			ctx, pageURL, bodyReading.LinkedURLs, refusals,
		),
	}, nil
}

func (reader *PageReader) contentTypeOf(ctx context.Context, declared string) ContentType {
	contentType, err := ContentTypeOf(declared)
	if err != nil {
		reader.observer.MediaTypeUnparsed(ctx, declared, err)
	}
	return contentType
}

func (reader *PageReader) readingOfBody(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	contentType ContentType,
	body []byte,
) (BodyReading, error) {
	for _, pageFormat := range reader.pageFormats {
		bodyReading, inPageFormat, err := pageFormat.ReadingOfBody(ctx, pageURL, contentType, body)
		if err != nil {
			return BodyReading{}, err
		}
		if inPageFormat {
			return bodyReading, nil
		}
	}
	return BodyReading{}, nil
}

func (reader *PageReader) discoveredURLsFrom(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	linkedURLs []canonicalurl.CanonicalURL,
	refusals robotsmeta.Refusals,
) []canonicalurl.CanonicalURL {
	if refusals.RefusesLinkDiscovery {
		reader.observer.LinkDiscoveryRefusalEnforced(ctx, pageURL)
		return nil
	}
	return linkedURLs
}
