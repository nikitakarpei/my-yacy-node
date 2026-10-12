package articlepermalinks_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlepermalinks"
)

func articleWithoutPermalinks(t *testing.T, article string) string {
	t.Helper()
	root, err := html.Parse(strings.NewReader(article))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	articlepermalinks.RemovePermalinks(root)
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

func TestRemovePermalinksRemovesAPilcrowThatLinksToItsParagraph(t *testing.T) {
	article := articleWithoutPermalinks(
		t,
		`<p id="section-1">TCP is reliable.<a href="#section-1">¶</a></p>`,
	)
	if article != `<p id="section-1">TCP is reliable.</p>` {
		t.Fatalf("permalink survived: %q", article)
	}
}

func TestRemovePermalinksRemovesAnIconThatLinksToItsSection(t *testing.T) {
	article := articleWithoutPermalinks(t,
		`<section id="usage"><h2>Usage<a href="#usage"><svg></svg></a></h2></section>`)
	if article != `<section id="usage"><h2>Usage</h2></section>` {
		t.Fatalf("permalink survived: %q", article)
	}
}

func TestRemovePermalinksKeepsALinkThatHoldsWords(t *testing.T) {
	page := `<div id="term-group"><p>Domain of <a href="#term-group">Group</a>.</p></div>`
	if article := articleWithoutPermalinks(t, page); article != page {
		t.Fatalf("a link with words went missing: %q", article)
	}
}

func TestRemovePermalinksKeepsAMarkThatLinksPastTheNearestElementWithAnID(t *testing.T) {
	page := `<div id="top"><div id="chapter"><p>Up: <a href="#top">↑</a></p></div></div>`
	if article := articleWithoutPermalinks(t, page); article != page {
		t.Fatalf("a link past the nearest element went missing: %q", article)
	}
}

func TestRemovePermalinksKeepsAMarkThatLinksElsewhereInThePage(t *testing.T) {
	page := `<p id="intro">See below <a href="#details">¶</a></p><p id="details">More.</p>`
	if article := articleWithoutPermalinks(t, page); article != page {
		t.Fatalf("a link elsewhere went missing: %q", article)
	}
}
