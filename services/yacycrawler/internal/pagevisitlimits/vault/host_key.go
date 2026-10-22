package vault

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

type hostKey struct {
	crawlID crawlid.CrawlID
	host    string
}

var hostKeys = vault.PairKey(vaultkeyparts.CrawlID, vault.TextKeyPart).KeyLayoutFor(
	func(key hostKey) (crawlid.CrawlID, string) { return key.crawlID, key.host },
	func(crawlID crawlid.CrawlID, host string) hostKey {
		return hostKey{crawlID: crawlID, host: host}
	},
)
