package pagehtmlreading

import (
	"iter"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type elementTree struct {
	root *html.Node
}

func (tree elementTree) elementsNamed(name atom.Atom) iter.Seq[element] {
	return func(yield func(element) bool) {
		var walk func(*html.Node) bool
		walk = func(node *html.Node) bool {
			if node.Type == html.ElementNode && node.DataAtom == name &&
				!yield(element{node: node}) {
				return false
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if !walk(child) {
					return false
				}
			}
			return true
		}
		walk(tree.root)
	}
}

type element struct {
	node *html.Node
}

func (e element) attributeOf(key string) (string, bool) {
	for _, attr := range e.node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val, true
		}
	}
	return "", false
}
