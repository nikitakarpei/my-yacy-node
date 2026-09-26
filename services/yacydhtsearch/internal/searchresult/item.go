// Package searchresult holds what the client reads of one search: one item per
// document, the ranking of those items, and the pages cut from the ranking;
// and the outcome that tells whether the peers were asked for the ranking.
package searchresult

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Item struct {
	Hash         yacymodel.URLHash
	Address      string
	Title        string
	Description  string
	PublishedAt  yacymodel.Optional[time.Time]
	ImageAddress string
}
