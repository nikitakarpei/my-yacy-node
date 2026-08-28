// Package pagerefusals reads what a page refuses to let a crawler do.
package pagerefusals

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

const (
	directiveNoIndex  = "noindex"
	directiveNoFollow = "nofollow"
	directiveNone     = "none"

	elementMeta = "meta"
)

type Refusals struct {
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}

type IgnoredRefusals struct {
	IndexingRefusal bool
}

func (refusals Refusals) HonoredBy(ignored IgnoredRefusals) Refusals {
	return Refusals{
		RefusesIndexing:      refusals.RefusesIndexing && !ignored.IndexingRefusal,
		RefusesLinkDiscovery: refusals.RefusesLinkDiscovery,
	}
}

func RefusalsOfPage(robotsDirectives []string, elementTree pagehtml.ElementTree) Refusals {
	var refusals Refusals
	for _, stated := range robotsDirectives {
		refusals.readDirectives(stated)
	}
	for element := range elementTree.ElementsNamed(elementMeta) {
		refusals.readMetaRobots(element)
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

func (refusals *Refusals) readMetaRobots(element pagehtml.Element) {
	name, ok := element.AttributeOf("name")
	if !ok || !strings.EqualFold(strings.TrimSpace(name), "robots") {
		return
	}
	content, ok := element.AttributeOf("content")
	if !ok {
		return
	}
	refusals.readDirectives(content)
}
