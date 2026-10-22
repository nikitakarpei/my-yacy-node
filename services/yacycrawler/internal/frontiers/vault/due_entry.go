package vault

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

var dueKeyParts = vault.TripleKey(
	vault.IntegerKeyPart,
	vaultkeyparts.CrawlID,
	vaultkeyparts.URLHash,
)

var dueEntries = dueKeyParts.KeyLayoutFor(
	func(entry dueEntry) (int64, crawlid.CrawlID, urlhash.URLHash) {
		return entry.due, entry.key.crawlID, entry.key.url
	},
	func(due int64, crawlID crawlid.CrawlID, url urlhash.URLHash) dueEntry {
		return dueEntry{due: due, key: pageVisitKey{crawlID: crawlID, url: url}}
	},
)

type dueEntry struct {
	due int64
	key pageVisitKey
}
