// Package htmltree holds a parsed html tree as the body of a document. The tree
// does not change after it is handed over: the body renders it on request and
// gives each reader a copy of its own to change.
package htmltree

import (
	"bytes"
	"fmt"
	"slices"

	"golang.org/x/net/html"
)

type Body struct {
	root *html.Node
}

func New(root *html.Node) Body {
	return Body{root: root}
}

func (body Body) Bytes() ([]byte, error) {
	var rendered bytes.Buffer
	if err := html.Render(&rendered, body.root); err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}
	return rendered.Bytes(), nil
}

func (body Body) CopiedTree() *html.Node {
	return copyOf(body.root)
}

func copyOf(node *html.Node) *html.Node {
	copied := &html.Node{
		Type:      node.Type,
		DataAtom:  node.DataAtom,
		Data:      node.Data,
		Namespace: node.Namespace,
		Attr:      slices.Clone(node.Attr),
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		copied.AppendChild(copyOf(child))
	}
	return copied
}
