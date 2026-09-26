// Package htmlmeta reads the robots rules an HTML page states in its robots
// meta tags.
package htmlmeta

import (
	"bytes"
	"io"
	"mime"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/net/html/charset"

	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
)

var htmlMediaTypes = map[string]struct{}{
	"text/html":             {},
	"application/xhtml+xml": {},
}

func RefusalsOf(contentType string, body []byte) robotsmeta.Refusals {
	if !isHTML(contentType) {
		return robotsmeta.Refusals{}
	}
	decoded, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		decoded = bytes.NewReader(body)
	}
	return refusalsOfTheRobotsMetaTagsIn(decoded)
}

func isHTML(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType, _, _ = strings.Cut(strings.ToLower(contentType), ";")
	}
	_, listed := htmlMediaTypes[strings.TrimSpace(mediaType)]
	return listed
}

func refusalsOfTheRobotsMetaTagsIn(page io.Reader) robotsmeta.Refusals {
	var refusals robotsmeta.Refusals
	tokenizer := html.NewTokenizer(page)
	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return refusals
		case html.StartTagToken, html.SelfClosingTagToken:
			if content, stated := robotsContentOf(tokenizer.Token()); stated {
				refusals = refusals.With(robotsmeta.RefusalsIn(content))
			}
		case html.TextToken, html.EndTagToken, html.CommentToken, html.DoctypeToken:
		}
	}
}

func robotsContentOf(tag html.Token) (string, bool) {
	if tag.DataAtom != atom.Meta {
		return "", false
	}
	name, named := attributeOf(tag, "name")
	if !named || !strings.EqualFold(strings.TrimSpace(name), "robots") {
		return "", false
	}
	return attributeOf(tag, "content")
}

func attributeOf(tag html.Token, key string) (string, bool) {
	for _, attribute := range tag.Attr {
		if strings.EqualFold(attribute.Key, key) {
			return attribute.Val, true
		}
	}
	return "", false
}
