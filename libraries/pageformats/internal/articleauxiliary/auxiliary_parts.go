// Package articleauxiliary finds the auxiliary parts of an article: the parts
// that belong to the page but are secondary to its text, such as notes.
package articleauxiliary

import (
	"strings"

	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlefootnotes"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/htmlflattening"
)

type Parts struct {
	parts []*html.Node
}

func PartsOf(article *html.Node) Parts {
	return Parts{parts: outermostOf(articlefootnotes.NotesOf(article))}
}

func outermostOf(parts []*html.Node) []*html.Node {
	isPart := make(map[*html.Node]bool, len(parts))
	for _, part := range parts {
		isPart[part] = true
	}
	var outermost []*html.Node
	for _, part := range parts {
		if !liesWithinAnyOf(part, isPart) {
			outermost = append(outermost, part)
		}
	}
	return outermost
}

func liesWithinAnyOf(part *html.Node, isPart map[*html.Node]bool) bool {
	for ancestor := range part.Ancestors() {
		if isPart[ancestor] {
			return true
		}
	}
	return false
}

func (auxiliaryParts Parts) Text() string {
	texts := make([]string, 0, len(auxiliaryParts.parts))
	for _, part := range auxiliaryParts.parts {
		texts = append(texts, htmlflattening.Flatten(part))
	}
	return strings.Join(texts, "\n")
}

func (auxiliaryParts Parts) Remove() {
	for _, part := range auxiliaryParts.parts {
		part.Parent.RemoveChild(part)
	}
}
