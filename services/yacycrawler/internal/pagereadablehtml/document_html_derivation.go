package pagereadablehtml

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() crawlcapability.PageContentFormat {
	return crawlcapability.PageContentFormatReadableHTML
}

func (DocumentHTMLDerivation) Derive(pageURL string, body []byte) ([]byte, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}
	parsedURL, _ := url.Parse(pageURL)

	article, err := readability.FromDocument(root, parsedURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", crawlcapability.ErrUnextractable, err)
	}
	if !hasReadableText(article.Node) {
		return nil, fmt.Errorf("%w: empty content", crawlcapability.ErrUnextractable)
	}
	var readable bytes.Buffer
	if err := article.RenderHTML(&readable); err != nil {
		return nil, fmt.Errorf("%w: %w", crawlcapability.ErrUnextractable, err)
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
