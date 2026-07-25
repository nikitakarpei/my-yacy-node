package pageabsorption

import "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"

type AbsorptionProgress interface {
	PageDisposed(reason disposal.Reason)
}
