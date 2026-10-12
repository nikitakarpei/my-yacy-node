package htmlflattening

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type footnoteMarks struct {
	markers   map[*html.Node]struct{}
	markerIDs map[string]struct{}
}

func footnoteMarksOf(root *html.Node) footnoteMarks {
	marks := footnoteMarks{markers: map[*html.Node]struct{}{}, markerIDs: map[string]struct{}{}}
	for node := range root.Descendants() {
		if isFootnoteMarker(node) {
			marks.add(node)
		}
	}
	return marks
}

func isFootnoteMarker(node *html.Node) bool {
	if node.DataAtom == atom.Sup {
		return holdsTextOnlyIn(node, isSamePageLink)
	}
	return isSamePageLink(node) && holdsTextOnlyIn(node, isSup)
}

func holdsTextOnlyIn(container *html.Node, isHolder func(*html.Node) bool) bool {
	holdsText := false
	for node := range container.Descendants() {
		if node.Type != html.TextNode || strings.TrimSpace(node.Data) == "" {
			continue
		}
		if !isHeldWithin(container, node, isHolder) {
			return false
		}
		holdsText = true
	}
	return holdsText
}

func isHeldWithin(container *html.Node, node *html.Node, isHolder func(*html.Node) bool) bool {
	for ancestor := range node.Ancestors() {
		if ancestor == container {
			return false
		}
		if isHolder(ancestor) {
			return true
		}
	}
	return false
}

func isSamePageLink(node *html.Node) bool {
	return node.DataAtom == atom.A && targetIDOf(node) != ""
}

func targetIDOf(link *html.Node) string {
	address := attributeOf(link, "href")
	if !strings.HasPrefix(address, "#") {
		return ""
	}
	return strings.TrimPrefix(address, "#")
}

func attributeOf(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

func isSup(node *html.Node) bool {
	return node.DataAtom == atom.Sup
}

func (marks footnoteMarks) add(marker *html.Node) {
	marks.markers[marker] = struct{}{}
	for node := range marker.Descendants() {
		marks.addID(node)
	}
	marks.addID(marker)
}

func (marks footnoteMarks) addID(node *html.Node) {
	if id := attributeOf(node, "id"); id != "" {
		marks.markerIDs[id] = struct{}{}
	}
}

func (marks footnoteMarks) contain(node *html.Node) bool {
	if _, marker := marks.markers[node]; marker {
		return true
	}
	if !isSamePageLink(node) {
		return false
	}
	_, backToAMarker := marks.markerIDs[targetIDOf(node)]
	return backToAMarker
}
