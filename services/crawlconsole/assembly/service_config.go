package crawlconsoleassembly

import (
	"net/url"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/webuistyle"
)

const crawlerCallDeadline = 10 * time.Second

type ServiceConfig struct {
	ListenAddr    string
	CrawlerAPIURL *url.URL
	TabBar        webuistyle.TabBar
	OpsAddr       string
}

func (ServiceConfig) CrawlerCallDeadline() time.Duration {
	return crawlerCallDeadline
}
