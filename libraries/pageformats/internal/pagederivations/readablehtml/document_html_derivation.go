package readablehtml

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type DocumentHTMLDerivation struct{}

func FromDocumentHTML() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatReadableHTML
}

func (DocumentHTMLDerivation) BodyFrom(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("parse html: %w", err)
	}
	article, err := readability.FromDocument(root, pageURL.WebAddress())
	if err != nil {
		return nil, false, fmt.Errorf("extract the readable article: %w", err)
	}
	if !hasReadableText(article.Node) {
		return nil, false, nil
	}
	markup, err := markupOf(article)
	if err != nil {
		return nil, false, err
	}
	return markup, true, nil
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

func markupOf(article readability.Article) ([]byte, error) {
	var markup bytes.Buffer
	if err := article.RenderHTML(&markup); err != nil {
		return nil, fmt.Errorf("render the readable html: %w", err)
	}
	return markup.Bytes(), nil
}
