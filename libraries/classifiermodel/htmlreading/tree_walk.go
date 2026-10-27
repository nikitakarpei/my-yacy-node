package htmlreading

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	dataAttributePrefix = "data-"
	hrefAttribute       = "href"
)

var hiddenElements = map[atom.Atom]bool{
	atom.Script:   true,
	atom.Style:    true,
	atom.Noscript: true,
	atom.Template: true,
	atom.Svg:      true,
	atom.Iframe:   true,
	atom.Object:   true,
}

type treeWalk struct {
	titleParts          []string
	textParts           []string
	hrefs               []string
	dataAttributeValues []string
	baseHRef            string
}

func walkOf(root *html.Node) treeWalk {
	var walk treeWalk
	walk.visit(root, false)
	return walk
}

func (w *treeWalk) visit(node *html.Node, hidden bool) {
	if node.Type == html.TextNode {
		w.readText(node, hidden)
	}
	if node.Type == html.ElementNode {
		w.readAttributes(node)
		hidden = hidden || hiddenElements[node.DataAtom]
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		w.visit(child, hidden)
	}
}

func (w *treeWalk) readText(text *html.Node, hidden bool) {
	switch {
	case hidden:
	case text.Parent.DataAtom == atom.Title:
		w.titleParts = append(w.titleParts, text.Data)
	default:
		w.textParts = append(w.textParts, text.Data)
	}
}

func (w *treeWalk) readAttributes(element *html.Node) {
	for _, attribute := range element.Attr {
		switch {
		case attribute.Val == "" || attribute.Namespace != "":
		case strings.HasPrefix(attribute.Key, dataAttributePrefix):
			w.dataAttributeValues = append(w.dataAttributeValues, attribute.Val)
		case attribute.Key != hrefAttribute:
		case element.DataAtom == atom.A:
			w.hrefs = append(w.hrefs, attribute.Val)
		case element.DataAtom == atom.Base && w.baseHRef == "":
			w.baseHRef = attribute.Val
		}
	}
}
