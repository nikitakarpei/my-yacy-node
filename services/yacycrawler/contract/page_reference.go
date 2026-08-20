package yacycrawlcontract

import "time"

type PageReference struct {
	CanonicalURL CanonicalURL `json:"CanonicalURL"`
	Title        string       `json:"Title"`
	CrawledAt    time.Time    `json:"CrawledAt"`
	Language     string       `json:"Language"`
}
