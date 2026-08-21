package yacycrawlcontract

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageReference struct {
	CanonicalURL canonicalurl.CanonicalURL `json:"CanonicalURL"`
	Title        string                    `json:"Title"`
	CrawledAt    time.Time                 `json:"CrawledAt"`
	Language     string                    `json:"Language"`
}
