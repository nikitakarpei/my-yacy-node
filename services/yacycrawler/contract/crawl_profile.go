package yacycrawlcontract

type CrawlScope int

const (
	ScopeWide CrawlScope = iota
	ScopeDomain
	ScopeSubpath
)

const MatchAll = ".*"

const UnlimitedPagesPerHost = -1

type CrawlProfile struct {
	Name                   string     `json:"Name"`
	Scope                  CrawlScope `json:"Scope"`
	URLMustMatch           string     `json:"URLMustMatch"`
	URLMustNotMatch        string     `json:"URLMustNotMatch"`
	MaxDepth               int        `json:"MaxDepth"`
	AllowQueryURLs         bool       `json:"AllowQueryURLs"`
	MaxPagesPerHost        int        `json:"MaxPagesPerHost"`
	IgnoresIndexingRefusal bool       `json:"IgnoresIndexingRefusal"`
}
