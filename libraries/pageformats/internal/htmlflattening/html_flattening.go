package htmlflattening

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
	if node.Type == html.ElementNode && !keepsTextJoined(node.DataAtom) {
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

func keepsTextJoined(name atom.Atom) bool {
	switch name {
	case atom.A, atom.Abbr, atom.B, atom.Bdi, atom.Bdo, atom.Cite, atom.Code,
		atom.Data, atom.Dfn, atom.Em, atom.I, atom.Kbd, atom.Mark, atom.Q,
		atom.Rp, atom.Rt, atom.Ruby, atom.S, atom.Samp, atom.Small, atom.Span,
		atom.Strong, atom.Sub, atom.Sup, atom.Time, atom.U, atom.Var:
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
