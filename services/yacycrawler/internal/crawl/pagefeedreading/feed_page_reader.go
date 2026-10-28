// Package pagefeedreading reads the URLs a fetched RSS or Atom feed links to: the site the
// feed belongs to and each of its items. A feed that breaks off keeps the links read before
// the break, and the break goes to the observer.
package pagefeedreading

import (
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"strings"

	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
)

const (
	mediaTextXML        = "text/xml"
	mediaApplicationXML = "application/xml"
	suffixXML           = "+xml"
)

type LinkResolution interface {
	BaseURLOf(
		ctx context.Context,
		enclosingBaseURL canonicalurl.CanonicalURL,
		statedBaseURL string,
	) canonicalurl.CanonicalURL
	URLsOf(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		links []linkresolution.Link,
	) []canonicalurl.CanonicalURL
}

type FeedPageReader struct {
	linkResolution LinkResolution
	observer       FeedReadingObserver
}

func NewFeedPageReader(
	linkResolution LinkResolution,
	observer FeedReadingObserver,
) *FeedPageReader {
	return &FeedPageReader{linkResolution: linkResolution, observer: observer}
}

func (reader *FeedPageReader) ReadingOfBody(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	contentType pagereading.ContentType,
	body []byte,
) (pagereading.BodyReading, bool, error) {
	if !isXML(contentType) {
		return pagereading.BodyReading{}, false, nil
	}
	decoder := xmlDecoderOf(contentType, body)
	root, found := rootElementOf(decoder)
	if !found {
		return pagereading.BodyReading{}, false, nil
	}
	feed, isFeed := feedDocumentFor(root.Name)
	if !isFeed {
		return pagereading.BodyReading{}, false, nil
	}
	if err := decoder.DecodeElement(feed, &root); err != nil {
		reader.observer.FeedUnreadable(ctx, pageURL, err)
	}
	return pagereading.BodyReading{
		LinkedURLs: reader.linkResolution.URLsOf(
			ctx, pageURL, feed.linksFrom(ctx, pageURL, reader.linkResolution),
		),
	}, true, nil
}

func isXML(contentType pagereading.ContentType) bool {
	mediaType := contentType.MediaType()
	return mediaType == mediaTextXML ||
		mediaType == mediaApplicationXML ||
		strings.HasSuffix(mediaType, suffixXML)
}

func xmlDecoderOf(contentType pagereading.ContentType, body []byte) *xml.Decoder {
	decoded, err := charset.NewReaderLabel(contentType.Charset(), bytes.NewReader(body))
	if err != nil {
		return lenientDecoderOf(bytes.NewReader(body), charset.NewReaderLabel)
	}
	return lenientDecoderOf(decoded, keptAsDecoded)
}

func lenientDecoderOf(
	input io.Reader,
	charsetReader func(label string, input io.Reader) (io.Reader, error),
) *xml.Decoder {
	decoder := xml.NewDecoder(input)
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity
	decoder.CharsetReader = charsetReader
	return decoder
}

func keptAsDecoded(_ string, input io.Reader) (io.Reader, error) {
	return input, nil
}

func rootElementOf(decoder *xml.Decoder) (xml.StartElement, bool) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return xml.StartElement{}, false
		}
		if root, ok := token.(xml.StartElement); ok {
			return root, true
		}
	}
}
