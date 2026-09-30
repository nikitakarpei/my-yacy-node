// Package htmlreading reads the title, the visible text, the markup and the
// links of an HTML page.
package htmlreading

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/transform"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/pagelinks"
)

type Reading struct {
	Address     canonicalurl.CanonicalURL
	Title       string
	VisibleText string
	HTML        string
	BodyBytes   int
	WebLinks    []canonicalurl.CanonicalURL
	DataLinks   []pagelinks.DataLink
}

func ReadingFrom(address canonicalurl.CanonicalURL, body []byte, contentType string) Reading {
	pageHTML := htmlFrom(contentType, body)
	walk := walkOf(treeOf(pageHTML))
	base := pagelinks.BaseFrom(address, walk.baseHRef)
	return Reading{
		Address:     address,
		Title:       collapsedTextOf(walk.titleParts),
		VisibleText: collapsedTextOf(walk.textParts),
		HTML:        pageHTML,
		BodyBytes:   len(body),
		WebLinks:    pagelinks.WebLinksFrom(walk.hrefs, base),
		DataLinks:   pagelinks.DataLinksFrom(walk.dataAttributeValues, base),
	}
}

func htmlFrom(contentType string, body []byte) string {
	encoding, _, _ := charset.DetermineEncoding(body, contentType)
	pageHTML, _, _ := transform.String(encoding.NewDecoder(), string(body))
	return pageHTML
}

func treeOf(pageHTML string) *html.Node {
	root, _ := html.Parse(strings.NewReader(pageHTML))
	return root
}

func collapsedTextOf(parts []string) string {
	var text strings.Builder
	for _, part := range parts {
		for word := range strings.FieldsSeq(part) {
			if text.Len() > 0 {
				text.WriteByte(' ')
			}
			text.WriteString(word)
		}
	}
	return text.String()
}
