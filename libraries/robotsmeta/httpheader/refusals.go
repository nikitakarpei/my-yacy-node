// Package httpheader reads the robots rules a page states in its X-Robots-Tag
// header values.
package httpheader

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
)

var directivesWithAValue = map[string]struct{}{
	"unavailable_after": {},
	"max-snippet":       {},
	"max-image-preview": {},
	"max-video-preview": {},
}

func RefusalsOf(robotsTagValues []string) robotsmeta.Refusals {
	var refusals robotsmeta.Refusals
	for _, robotsTagValue := range robotsTagValues {
		if !namesABot(robotsTagValue) {
			refusals = refusals.With(robotsmeta.RefusalsIn(robotsTagValue))
		}
	}
	return refusals
}

func namesABot(robotsTagValue string) bool {
	leadingName, _, named := strings.Cut(robotsTagValue, ":")
	if !named || strings.Contains(leadingName, ",") {
		return false
	}
	_, isADirective := directivesWithAValue[strings.ToLower(strings.TrimSpace(leadingName))]
	return !isADirective
}
