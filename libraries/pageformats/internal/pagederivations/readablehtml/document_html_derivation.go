package readablehtml

import (
	"bytes"
	"fmt"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

type DocumentHTMLDerivation struct{}

func NewDocumentHTMLDerivation() DocumentHTMLDerivation {
	return DocumentHTMLDerivation{}
}

func (DocumentHTMLDerivation) SourceFormat() documentextraction.Format {
	return documentextraction.FormatDocumentHTML
}

func (DocumentHTMLDerivation) TargetFormat() documentextraction.Format {
	return documentextraction.FormatReadableHTML
}

func (DocumentHTMLDerivation) Derive(
	pageURL canonicalurl.CanonicalURL,
	body []byte,
) ([]byte, bool, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("parse html: %w", err)
	}
	article, readable := readableArticleOf(root, pageURL)
	if !readable {
		return nil, false, nil
	}
	markup, rendered := markupOf(article)
	return markup, rendered, nil
}

func readableArticleOf(
	root *html.Node,
	pageURL canonicalurl.CanonicalURL,
) (readability.Article, bool) {
	article, err := readability.FromDocument(root, pageURL.WebAddress())
	if err != nil {
		return readability.Article{}, false
	}
	return article, hasReadableText(article.Node)
}

func markupOf(article readability.Article) ([]byte, bool) {
	var markup bytes.Buffer
	if err := article.RenderHTML(&markup); err != nil {
		return nil, false
	}
	return markup.Bytes(), true
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
