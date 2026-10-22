package crawlprogress

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	crawlProgressBucket = "crawl_progress"
	hostProgressBucket  = "crawl_host_progress"
)

func registerCounts(storage *vault.Vault) (*Counts, error) {
	ofCrawl, err := storage.RegisterAmounts(crawlProgressBucket, crawlCounts)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", crawlProgressBucket, err)
	}
	ofHost, err := storage.RegisterAmounts(hostProgressBucket, hostCounts)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", hostProgressBucket, err)
	}
	return &Counts{ofCrawl: ofCrawl, ofHost: ofHost}, nil
}

type crawlCount struct {
	crawlID crawlid.CrawlID
	count   string
}

type hostCount struct {
	crawlID crawlid.CrawlID
	host    string
	count   string
}

var crawlCountKeyParts = vault.PairKey(vaultkeyparts.CrawlID, vault.TextKeyPart)

var crawlCounts = crawlCountKeyParts.KeyLayoutFor(
	func(key crawlCount) (crawlid.CrawlID, string) { return key.crawlID, key.count },
	func(crawlID crawlid.CrawlID, count string) crawlCount {
		return crawlCount{crawlID: crawlID, count: count}
	},
)

var hostCountKeyParts = vault.TripleKey(vaultkeyparts.CrawlID, vault.TextKeyPart, vault.TextKeyPart)

var hostCounts = hostCountKeyParts.KeyLayoutFor(
	func(key hostCount) (crawlid.CrawlID, string, string) { return key.crawlID, key.host, key.count },
	func(crawlID crawlid.CrawlID, host string, count string) hostCount {
		return hostCount{crawlID: crawlID, host: host, count: count}
	},
)
