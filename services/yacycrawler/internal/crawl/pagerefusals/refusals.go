// Package pagerefusals reads what a page refuses to let a crawler do.
package pagerefusals

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
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

func RefusalsOfPage(robotsDirectives []string, elementTree pagehtml.ElementTree) Refusals {
	var refusals Refusals
	for _, stated := range robotsDirectives {
		refusals.readDirectives(stated)
	}
	for node := range elementTree.Elements() {
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
	name, ok := pagehtml.AttributeOf(node, "name")
	if !ok || !strings.EqualFold(strings.TrimSpace(name), "robots") {
		return
	}
	content, ok := pagehtml.AttributeOf(node, "content")
	if !ok {
		return
	}
	refusals.readDirectives(content)
}
