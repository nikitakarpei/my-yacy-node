package yacycrawlcontract

type CrawlScope string

const (
	ScopeWide    CrawlScope = "wide"
	ScopeDomain  CrawlScope = "domain"
	ScopeSubpath CrawlScope = "subpath"
)

var CrawlScopes = []CrawlScope{ScopeWide, ScopeDomain, ScopeSubpath}

const MatchAll = ".*"

const UnlimitedPagesPerHost = 0

type CrawlProfile struct {
	Scope           CrawlScope   `json:"Scope"`
	URLMustMatch    string       `json:"URLMustMatch,omitzero"`
	URLMustNotMatch string       `json:"URLMustNotMatch,omitzero"`
	MaxDepth        int          `json:"MaxDepth"`
	AllowQueryURLs  bool         `json:"AllowQueryURLs,omitzero"`
	MaxPagesPerHost int          `json:"MaxPagesPerHost,omitzero"`
	RecrawlGrace    RecrawlGrace `json:"RecrawlGrace,omitzero"`
}
