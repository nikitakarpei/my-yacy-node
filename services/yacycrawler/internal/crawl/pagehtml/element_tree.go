package pagehtml

import (
	"iter"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type ElementTree struct {
	root *html.Node
}

func (t ElementTree) ElementsNamed(name string) iter.Seq[Element] {
	return func(yield func(Element) bool) {
		if t.root == nil {
			return
		}
		wanted := atom.Lookup([]byte(strings.ToLower(name)))
		var walk func(*html.Node) bool
		walk = func(node *html.Node) bool {
			if isNamed(node, name, wanted) && !yield(Element{node: node}) {
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

func isNamed(node *html.Node, name string, wanted atom.Atom) bool {
	if node.Type != html.ElementNode {
		return false
	}
	if wanted != 0 {
		return node.DataAtom == wanted
	}
	return strings.EqualFold(node.Data, name)
}

type Element struct {
	node *html.Node
}

func (e Element) AttributeOf(key string) (string, bool) {
	for _, attr := range e.node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val, true
		}
	}
	return "", false
}
