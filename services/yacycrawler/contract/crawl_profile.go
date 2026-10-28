package yacycrawlcontract

import "time"

type CrawlScope string

const (
	ScopeWide    CrawlScope = "wide"
	ScopeDomain  CrawlScope = "domain"
	ScopeSubpath CrawlScope = "subpath"
)

var CrawlScopes = []CrawlScope{ScopeWide, ScopeDomain, ScopeSubpath}

const MatchAll = ".*"

const UnlimitedPagesPerHost = 0

const (
	NoRecrawlGrace  time.Duration = 0
	MaxRecrawlGrace               = 30 * 24 * time.Hour
)

type CrawlProfile struct {
	Scope           CrawlScope    `json:"Scope"`
	URLMustMatch    string        `json:"URLMustMatch,omitzero"`
	URLMustNotMatch string        `json:"URLMustNotMatch,omitzero"`
	MaxDepth        int           `json:"MaxDepth"`
	AllowQueryURLs  bool          `json:"AllowQueryURLs,omitzero"`
	MaxPagesPerHost int           `json:"MaxPagesPerHost,omitzero"`
	RecrawlGrace    time.Duration `json:"RecrawlGrace,omitzero"`
}
