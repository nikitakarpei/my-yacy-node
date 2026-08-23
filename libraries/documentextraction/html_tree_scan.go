package documentextraction

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type treeScan struct {
	title    string
	language string
}

func scanTree(root *html.Node) treeScan {
	var scan treeScan
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			scan.inspect(node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return scan
}

func (scan *treeScan) inspect(node *html.Node) {
	switch node.DataAtom {
	case atom.Html:
		if lang, ok := attribute(node, "lang"); ok && scan.language == "" {
			scan.language = lang
		}
	case atom.Title:
		if scan.title == "" {
			scan.title = strings.TrimSpace(textContent(node))
		}
	}
}

func textContent(node *html.Node) string {
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			builder.WriteString(child.Data)
		}
	}
	return builder.String()
}

func attribute(node *html.Node, key string) (string, bool) {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val, true
		}
	}
	return "", false
}
