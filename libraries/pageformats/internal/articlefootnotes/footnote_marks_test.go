package articlefootnotes_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlefootnotes"
)

func articleWithoutFootnoteMarks(t *testing.T, article string) string {
	t.Helper()
	root, err := html.Parse(strings.NewReader(article))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	articlefootnotes.RemoveFootnoteMarks(root)
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

func TestRemoveFootnoteMarksRemovesAMarkerAndTheLinkBackToIt(t *testing.T) {
	article := articleWithoutFootnoteMarks(t,
		`<p>TCP.<sup id="ref-1"><a href="#note-1">[1]</a></sup></p>`+
			`<ol><li id="note-1"><a href="#ref-1">↑</a> Stevens.</li></ol>`)
	if article != `<p>TCP.</p><ol><li id="note-1"> Stevens.</li></ol>` {
		t.Fatalf("footnote marks survived: %q", article)
	}
}

func TestRemoveFootnoteMarksRemovesAMarkerThatALinkAroundASupMakes(t *testing.T) {
	article := articleWithoutFootnoteMarks(t,
		`<p>TCP.<a href="#fn1" id="fnref1"><sup>1</sup></a></p>`+
			`<ol><li id="fn1">Stevens. <a href="#fnref1">↩︎</a></li></ol>`)
	if article != `<p>TCP.</p><ol><li id="fn1">Stevens. </li></ol>` {
		t.Fatalf("footnote marks survived: %q", article)
	}
}

func TestRemoveFootnoteMarksKeepsASupThatIsNoFootnoteMarker(t *testing.T) {
	page := `<p>About 10<sup>6</sup> pages, see <sup><a href="https://example.org/">source</a></sup>.</p>`
	if article := articleWithoutFootnoteMarks(t, page); article != page {
		t.Fatalf("a plain sup went missing: %q", article)
	}
}

func TestRemoveFootnoteMarksKeepsALinkToASectionOfTheArticle(t *testing.T) {
	page := `<ul><li><a href="#history">History</a></li></ul><h2 id="history">History</h2>`
	if article := articleWithoutFootnoteMarks(t, page); article != page {
		t.Fatalf("a link to a section went missing: %q", article)
	}
}
