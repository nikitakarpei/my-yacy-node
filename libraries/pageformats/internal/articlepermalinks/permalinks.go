// Package articlepermalinks finds the permalinks of an article: the marks,
// such as a pilcrow or an icon, that link to the element they sit in.
package articlepermalinks

import (
	"strings"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func RemovePermalinks(article *html.Node) {
	var permalinks []*html.Node
	for node := range article.Descendants() {
		if isPermalink(node) {
			permalinks = append(permalinks, node)
		}
	}
	for _, permalink := range permalinks {
		permalink.Parent.RemoveChild(permalink)
	}
}

func isPermalink(node *html.Node) bool {
	if node.DataAtom != atom.A {
		return false
	}
	targetID, linksWithinThePage := strings.CutPrefix(attributeOf(node, "href"), "#")
	return linksWithinThePage &&
		targetID != "" &&
		targetID == nearestAncestorIDOf(node) &&
		holdsNoWords(node)
}

func attributeOf(node *html.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

func nearestAncestorIDOf(node *html.Node) string {
	for ancestor := range node.Ancestors() {
		if id := attributeOf(ancestor, "id"); id != "" {
			return id
		}
	}
	return ""
}

func holdsNoWords(node *html.Node) bool {
	for descendant := range node.Descendants() {
		if descendant.Type == html.TextNode &&
			strings.ContainsFunc(descendant.Data, isWordCharacter) {
			return false
		}
	}
	return true
}

func isWordCharacter(character rune) bool {
	return unicode.IsLetter(character) || unicode.IsDigit(character)
}
