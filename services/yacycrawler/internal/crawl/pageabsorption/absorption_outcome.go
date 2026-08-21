package pageabsorption

import (
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
)

type AbsorptionOutcome struct {
	DiscoveredURLs []canonicalurl.CanonicalURL
	Disposal       disposal.Reason
}
