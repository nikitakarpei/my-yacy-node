package htmltext

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func Flatten(body []byte) (string, error) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}
	var text strings.Builder
	writeText(&text, root)
	return collapseWhitespace(text.String()), nil
}

func writeText(text *strings.Builder, node *html.Node) {
	if node.Type == html.TextNode {
		text.WriteString(strings.ReplaceAll(node.Data, "\n", " "))
		return
	}
	if node.Type == html.ElementNode && isSkipped(node.DataAtom) {
		return
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		writeText(text, child)
	}
	if node.Type == html.ElementNode && isBlock(node.DataAtom) {
		text.WriteString("\n")
	}
}

func isSkipped(name atom.Atom) bool {
	switch name {
	case atom.Script, atom.Style, atom.Noscript, atom.Template:
		return true
	}
	return false
}

func isBlock(name atom.Atom) bool {
	switch name {
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Header, atom.Footer,
		atom.Aside, atom.Nav, atom.Main, atom.Blockquote, atom.Pre, atom.Figure,
		atom.Figcaption, atom.Ul, atom.Ol, atom.Li, atom.Dl, atom.Dt, atom.Dd,
		atom.Table, atom.Tr, atom.Td, atom.Th, atom.Br, atom.Hr,
		atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		return true
	}
	return false
}

func collapseWhitespace(text string) string {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if fields := strings.Fields(line); len(fields) > 0 {
			kept = append(kept, strings.Join(fields, " "))
		}
	}
	return strings.Join(kept, "\n")
}
