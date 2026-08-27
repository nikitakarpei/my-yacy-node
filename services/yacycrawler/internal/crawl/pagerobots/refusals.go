// Package pagerobots reads what a page refuses to let a crawler do.
package pagerobots

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagemarkup"
)

const (
	directiveNoIndex  = "noindex"
	directiveNoFollow = "nofollow"
	directiveNone     = "none"
)

type Refusals struct {
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}

func RefusalsOfPage(statedDirectives []string, markup pagemarkup.Markup) Refusals {
	var refusals Refusals
	for _, stated := range statedDirectives {
		refusals.readDirectives(stated)
	}
	for node := range markup.Elements() {
		if node.DataAtom == atom.Meta {
			refusals.readMetaRobots(node)
		}
	}
	return refusals
}

func (refusals *Refusals) readDirectives(stated string) {
	for _, directive := range strings.Split(stated, ",") {
		switch strings.ToLower(strings.TrimSpace(directive)) {
		case directiveNoIndex:
			refusals.RefusesIndexing = true
		case directiveNoFollow:
			refusals.RefusesLinkDiscovery = true
		case directiveNone:
			refusals.RefusesIndexing = true
			refusals.RefusesLinkDiscovery = true
		}
	}
}

func (refusals *Refusals) readMetaRobots(node *html.Node) {
	name, ok := pagemarkup.AttributeOf(node, "name")
	if !ok || !strings.EqualFold(strings.TrimSpace(name), "robots") {
		return
	}
	content, ok := pagemarkup.AttributeOf(node, "content")
	if !ok {
		return
	}
	refusals.readDirectives(content)
}
