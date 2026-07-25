package frontier

import "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"

type Config struct {
	MaxDeferralsPerURL int
	MaxAttemptsPerURL  int
	RetryDelay         retrydelay.Bounds
}
