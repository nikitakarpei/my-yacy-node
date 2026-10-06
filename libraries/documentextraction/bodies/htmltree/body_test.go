package htmltree_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction/bodies/htmltree"
)

func parsedTree(t *testing.T, page string) *html.Node {
	t.Helper()
	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return root
}

func renderedBytesOf(t *testing.T, body htmltree.Body) string {
	t.Helper()
	rendered, err := body.Bytes()
	if err != nil {
		t.Fatalf("bytes: %v", err)
	}
	return string(rendered)
}

func articleOf(root *html.Node) *html.Node {
	if root.Type == html.ElementNode && root.Data == "article" {
		return root
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if article := articleOf(child); article != nil {
			return article
		}
	}
	return nil
}

func TestBytesRenderTheHeldTree(t *testing.T) {
	body := htmltree.New(parsedTree(t, `<p class="lead">first</p>`))

	rendered := renderedBytesOf(t, body)

	want := `<html><head></head><body><p class="lead">first</p></body></html>`
	if rendered != want {
		t.Fatalf("rendered %q, want %q", rendered, want)
	}
}

func TestAChangedCopyOfASubtreeLeavesTheHeldTreeUnchanged(t *testing.T) {
	article := articleOf(parsedTree(t,
		`<nav>menu</nav><article><p class="lead">first</p></article>`+
			`<footer>end</footer>`,
	))
	body := htmltree.New(article)
	before := renderedBytesOf(t, body)

	copied := body.CopiedTree()
	copied.FirstChild.Attr[0].Val = "changed"
	copied.FirstChild.FirstChild.Data = "changed"
	copied.AppendChild(&html.Node{Type: html.TextNode, Data: "added"})

	if after := renderedBytesOf(t, body); after != before {
		t.Fatalf("the held tree changed from %q to %q", before, after)
	}
}

func TestACopyIsDetachedFromTheTreeAroundIt(t *testing.T) {
	article := articleOf(parsedTree(t, `<nav>menu</nav><article><p>first</p></article>`))

	copied := htmltree.New(article).CopiedTree()

	if copied.Parent != nil || copied.PrevSibling != nil || copied.NextSibling != nil {
		t.Fatal("the copy still points at the tree around it")
	}
}

func TestACopyRendersLikeTheHeldTree(t *testing.T) {
	body := htmltree.New(parsedTree(t, `<p>first</p><svg><style>a&lt;b</style></svg>`))

	copied := htmltree.New(body.CopiedTree())

	if rendered, want := renderedBytesOf(t, copied), renderedBytesOf(t, body); rendered != want {
		t.Fatalf("the copy rendered %q, want %q", rendered, want)
	}
}

func TestAVoidElementHoldingChildrenInSVGFailsToRender(t *testing.T) {
	body := htmltree.New(parsedTree(t, `<svg><source>x</source></svg>`))

	if _, err := body.Bytes(); err == nil {
		t.Fatal("a void element with children rendered")
	}
}
