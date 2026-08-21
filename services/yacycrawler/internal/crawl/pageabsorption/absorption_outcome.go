package pageabsorption

import "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"

type AbsorptionOutcome struct {
	DiscoveredURLs []string
	Disposal       disposal.Reason
}
