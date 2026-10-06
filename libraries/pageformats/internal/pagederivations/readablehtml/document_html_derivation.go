package readablehtml

import (
	"context"
	"fmt"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
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
	source documentextraction.Body,
) (documentextraction.Body, bool, error) {
	parser := readability.NewParser()
	article, err := parser.ParseAndMutate(
		source.(htmltree.Body).CopiedTree(), pageURL.WebAddress(),
	)
	if err != nil {
		return nil, false, fmt.Errorf("extract the readable article: %w", err)
	}
	if !hasReadableText(article.Node) {
		return nil, false, nil
	}
	alignAtomsWithTagNames(article.Node)
	return htmltree.New(article.Node), true, nil
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

func alignAtomsWithTagNames(node *html.Node) {
	if node.Type == html.ElementNode && node.Namespace == "" {
		node.DataAtom = atom.Lookup([]byte(node.Data))
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		alignAtomsWithTagNames(child)
	}
}
