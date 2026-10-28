// Package pagehtmlreading reads what a fetched HTML page states in its markup: the
// refusals of its robots meta tags and the URLs its anchors link to.
package pagehtmlreading

import (
	"bytes"
	"context"
	"fmt"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/htmlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkresolution"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagereading"
)

const (
	mediaHTML     = "text/html"
	mediaXHTML    = "application/xhtml+xml"
	attributeHref = "href"
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

type HTMLPageReader struct {
	linkResolution LinkResolution
}

func NewHTMLPageReader(linkResolution LinkResolution) *HTMLPageReader {
	return &HTMLPageReader{linkResolution: linkResolution}
}

func (reader *HTMLPageReader) ReadingOfBody(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	contentType pagereading.ContentType,
	body []byte,
) (pagereading.BodyReading, bool, error) {
	if !isHTML(contentType) {
		return pagereading.BodyReading{}, false, nil
	}
	tree, err := elementTreeFrom(contentType, body)
	if err != nil {
		return pagereading.BodyReading{}, true, err
	}
	return pagereading.BodyReading{
		Refusals:   htmlmeta.RefusalsOf(contentType.String(), body),
		LinkedURLs: reader.linkedURLsFrom(ctx, tree, pageURL),
	}, true, nil
}

func isHTML(contentType pagereading.ContentType) bool {
	return contentType.MediaType() == mediaHTML || contentType.MediaType() == mediaXHTML
}

func elementTreeFrom(contentType pagereading.ContentType, body []byte) (elementTree, error) {
	decoded, err := charset.NewReader(bytes.NewReader(body), contentType.String())
	if err != nil {
		return elementTree{}, fmt.Errorf("read html charset: %w", err)
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return elementTree{}, fmt.Errorf("parse html: %w", err)
	}
	return elementTree{root: root}, nil
}

func (reader *HTMLPageReader) linkedURLsFrom(
	ctx context.Context,
	tree elementTree,
	pageURL canonicalurl.CanonicalURL,
) []canonicalurl.CanonicalURL {
	baseURL := reader.linkResolution.BaseURLOf(ctx, pageURL, baseHrefOf(tree))
	return reader.linkResolution.URLsOf(ctx, pageURL, anchorLinksOf(tree, baseURL))
}

func baseHrefOf(tree elementTree) string {
	for base := range tree.elementsNamed(atom.Base) {
		if href, ok := base.attributeOf(attributeHref); ok {
			return href
		}
	}
	return ""
}

func anchorLinksOf(tree elementTree, baseURL canonicalurl.CanonicalURL) []linkresolution.Link {
	var links []linkresolution.Link
	for anchor := range tree.elementsNamed(atom.A) {
		if href, ok := anchor.attributeOf(attributeHref); ok {
			links = append(links, linkresolution.Link{BaseURL: baseURL, Reference: href})
		}
	}
	return links
}
