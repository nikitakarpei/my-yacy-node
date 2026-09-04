// Package pagehtml parses a fetched body into the HTML elements the crawler reads.
package pagehtml

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	mediaHTML  = "text/html"
	mediaXHTML = "application/xhtml+xml"
)

var ErrNotHTML = errors.New("not an html page")

type HTMLParser struct {
	observer MediaTypeObserver
}

func NewHTMLParser(observer MediaTypeObserver) *HTMLParser {
	return &HTMLParser{observer: observer}
}

func (parser *HTMLParser) ElementTreeFrom(
	ctx context.Context,
	contentType string,
	body []byte,
) (ElementTree, error) {
	if !parser.isHTML(ctx, contentType) {
		return ElementTree{}, ErrNotHTML
	}
	decoded, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return ElementTree{}, fmt.Errorf("decode charset: %w", err)
	}
	root, err := html.Parse(decoded)
	if err != nil {
		return ElementTree{}, fmt.Errorf("parse html: %w", err)
	}
	return ElementTree{root: root}, nil
}

func (parser *HTMLParser) isHTML(ctx context.Context, contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		parser.observer.MediaTypeUnparsed(ctx, contentType, err)
		media = strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	}
	return media == mediaHTML || media == mediaXHTML
}
