package articleauxiliary_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articleauxiliary"
)

func parsed(t *testing.T, article string) *html.Node {
	t.Helper()
	root, err := html.Parse(strings.NewReader(article))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return root
}

func renderedBodyOf(t *testing.T, root *html.Node) string {
	t.Helper()
	var rendered strings.Builder
	for node := range root.Descendants() {
		if node.DataAtom == atom.Body {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if err := html.Render(&rendered, child); err != nil {
					t.Fatalf("render: %v", err)
				}
			}
		}
	}
	return rendered.String()
}

const articleWithTwoNotes = `<p>TCP.<sup><a href="#note-1">[1]</a></sup>` +
	`<sup><a href="#note-2">[2]</a></sup></p>` +
	`<ol><li id="note-1">Stevens.</li><li id="note-2">Postel.</li></ol>`

func TestTextHoldsEachNoteOnItsOwnLine(t *testing.T) {
	auxiliaryParts := articleauxiliary.PartsOf(parsed(t, articleWithTwoNotes))
	if text := auxiliaryParts.Text(); text != "Stevens.\nPostel." {
		t.Fatalf("auxiliary text = %q", text)
	}
}

func TestTextHoldsANoteWithinANoteOnce(t *testing.T) {
	auxiliaryParts := articleauxiliary.PartsOf(parsed(t,
		`<p>TCP.<sup><a href="#note-a">[a]</a></sup><sup><a href="#note-1">[1]</a></sup></p>`+
			`<ol><li id="note-a">See <cite id="note-1">Stevens.</cite></li></ol>`))
	if text := auxiliaryParts.Text(); text != "See Stevens." {
		t.Fatalf("auxiliary text = %q", text)
	}
}

func TestRemoveLeavesTheArticleWithoutItsNotes(t *testing.T) {
	article := parsed(t, articleWithTwoNotes)
	articleauxiliary.PartsOf(article).Remove()
	if rendered := renderedBodyOf(t, article); rendered !=
		`<p>TCP.<sup><a href="#note-1">[1]</a></sup><sup><a href="#note-2">[2]</a></sup></p><ol></ol>` {
		t.Fatalf("notes survived: %q", rendered)
	}
}
