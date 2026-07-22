package readablehtml

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() contentformatgraph.Format {
	return contentformatgraph.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableHTML
}

func (DocumentHTMLDerivation) Derive(pageURL string, body []byte) ([]byte, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	parsedURL, _ := url.Parse(pageURL)

	article, err := readability.FromDocument(root, parsedURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", contentformatgraph.ErrUnextractable, err)
	}
	if !hasReadableText(article.Node) {
		return nil, fmt.Errorf("%w: empty content", contentformatgraph.ErrUnextractable)
	}
	var readable bytes.Buffer
	if err := article.RenderHTML(&readable); err != nil {
		return nil, fmt.Errorf("%w: %w", contentformatgraph.ErrUnextractable, err)
	}
	return readable.Bytes(), nil
}

func hasReadableText(node *html.Node) bool {
	if node == nil {
		return false
	}
	if node.Type == html.TextNode && strings.TrimSpace(node.Data) != "" {
		return true
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if hasReadableText(child) {
			return true
		}
	}
	return false
}
