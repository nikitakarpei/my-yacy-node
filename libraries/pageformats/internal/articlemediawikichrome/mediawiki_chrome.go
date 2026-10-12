// Package articlemediawikichrome finds the MediaWiki chrome of an article:
// the edit links and navigation boxes that MediaWiki marks by class.
package articlemediawikichrome

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
)

var mediaWikiChromeClasses = []string{
	"mw-editsection",
	"navbox",
}

func RemoveMediaWikiChrome(article *html.Node) {
	var chrome []*html.Node
	for node := range article.Descendants() {
		if isMediaWikiChrome(node) {
			chrome = append(chrome, node)
		}
	}
	for _, element := range chrome {
		element.Parent.RemoveChild(element)
	}
}

func isMediaWikiChrome(node *html.Node) bool {
	return slices.ContainsFunc(classesOf(node), func(class string) bool {
		return slices.Contains(mediaWikiChromeClasses, class)
	})
}

func classesOf(node *html.Node) []string {
	for _, attribute := range node.Attr {
		if attribute.Key == "class" {
			return strings.Fields(attribute.Val)
		}
	}
	return nil
}
