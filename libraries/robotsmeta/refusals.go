// Package robotsmeta holds what the robots rules a page states about itself
// refuse to let a search engine do, and reads a list of those rules. Its
// subpackages read the rules from one carrier each.
package robotsmeta

import "strings"

const (
	ruleNoIndex  = "noindex"
	ruleNoFollow = "nofollow"
	ruleNone     = "none"
)

type Refusals struct {
	RefusesIndexing      bool
	RefusesLinkDiscovery bool
}

func RefusalsIn(ruleList string) Refusals {
	var refusals Refusals
	for rule := range strings.SplitSeq(ruleList, ",") {
		switch strings.ToLower(strings.TrimSpace(rule)) {
		case ruleNoIndex:
			refusals.RefusesIndexing = true
		case ruleNoFollow:
			refusals.RefusesLinkDiscovery = true
		case ruleNone:
			refusals.RefusesIndexing = true
			refusals.RefusesLinkDiscovery = true
		}
	}
	return refusals
}

func (refusals Refusals) With(otherRefusals Refusals) Refusals {
	return Refusals{
		RefusesIndexing:      refusals.RefusesIndexing || otherRefusals.RefusesIndexing,
		RefusesLinkDiscovery: refusals.RefusesLinkDiscovery || otherRefusals.RefusesLinkDiscovery,
	}
}
