// Package pagehtml parses a fetched body into the HTML elements the crawler reads.
package pagehtml

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"mime"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	mediaHTML  = "text/html"
	mediaXHTML = "application/xhtml+xml"

	msgMediaTypeUnparsed = "content type unparsed, falling back to its leading segment"
)

var ErrNotHTML = errors.New("not an html page")

type ElementTree struct {
	root *html.Node
}

func ElementTreeFrom(ctx context.Context, contentType string, body []byte) (ElementTree, error) {
	if !isHTML(ctx, contentType) {
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

func isHTML(ctx context.Context, contentType string) bool {
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		slog.DebugContext(ctx, msgMediaTypeUnparsed,
			slog.String("contentType", contentType),
			slog.Any("error", err),
		)
		media = strings.ToLower(strings.TrimSpace(strings.SplitN(contentType, ";", 2)[0]))
	}
	return media == mediaHTML || media == mediaXHTML
}

func (t ElementTree) Elements() iter.Seq[*html.Node] {
	return func(yield func(*html.Node) bool) {
		if t.root == nil {
			return
		}
		var walk func(*html.Node) bool
		walk = func(node *html.Node) bool {
			if node.Type == html.ElementNode && !yield(node) {
				return false
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if !walk(child) {
					return false
				}
			}
			return true
		}
		walk(t.root)
	}
}

func AttributeOf(node *html.Node, key string) (string, bool) {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val, true
		}
	}
	return "", false
}
