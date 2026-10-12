package articlemediawikichrome_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/pageformats/internal/articlemediawikichrome"
)

func articleWithoutMediaWikiChrome(t *testing.T, article string) string {
	t.Helper()
	root, err := html.Parse(strings.NewReader(article))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	articlemediawikichrome.RemoveMediaWikiChrome(root)
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

func TestRemoveMediaWikiChromeRemovesAnEditLink(t *testing.T) {
	article := articleWithoutMediaWikiChrome(t,
		`<h2>History<span class="mw-editsection">[<a href="/edit">edit</a>]</span></h2>`)
	if article != `<h2>History</h2>` {
		t.Fatalf("edit link survived: %q", article)
	}
}

func TestRemoveMediaWikiChromeRemovesANavigationBox(t *testing.T) {
	article := articleWithoutMediaWikiChrome(t,
		`<p>TCP is reliable.</p><div class="navbox authority-control"><a href="/udp">UDP</a></div>`)
	if article != `<p>TCP is reliable.</p>` {
		t.Fatalf("navigation box survived: %q", article)
	}
}

func TestRemoveMediaWikiChromeKeepsACollapsedSuccessionBox(t *testing.T) {
	page := `<table class="wikitable succession-box mw-collapsible autocollapse">` +
		`<tbody><tr><td>Preceded by New York Giants</td></tr></tbody></table>`
	if article := articleWithoutMediaWikiChrome(t, page); article != page {
		t.Fatalf("succession box went missing: %q", article)
	}
}

func TestRemoveMediaWikiChromeKeepsAClassThatOnlyStartsLikeChrome(t *testing.T) {
	page := `<div class="navboxes-of-the-author"><p>TCP is reliable.</p></div>`
	if article := articleWithoutMediaWikiChrome(t, page); article != page {
		t.Fatalf("article text went missing: %q", article)
	}
}
