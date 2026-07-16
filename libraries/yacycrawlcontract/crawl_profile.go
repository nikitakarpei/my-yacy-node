package yacycrawlcontract

import (
	"crypto/md5"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type CrawlScope int

const (
	ScopeWide CrawlScope = iota
	ScopeDomain
	ScopeSubpath
)

const MatchAll = ".*"

const UnlimitedPagesPerHost = -1

type CrawlProfile struct {
	Handle          string        `json:"Handle"`
	Name            string        `json:"Name"`
	Scope           CrawlScope    `json:"Scope"`
	URLMustMatch    string        `json:"URLMustMatch"`
	URLMustNotMatch string        `json:"URLMustNotMatch"`
	MaxDepth        int           `json:"MaxDepth"`
	AllowQueryURLs  bool          `json:"AllowQueryURLs"`
	MaxPagesPerHost int           `json:"MaxPagesPerHost"`
	CrawlDelay      time.Duration `json:"CrawlDelay"`
}

func NewCrawlProfile(profile CrawlProfile) CrawlProfile {
	profile.Handle = profile.ComputeHandle()
	return profile
}

func (p CrawlProfile) ComputeHandle() string {
	raw := fmt.Sprintf(
		"%s\x00%s\x00%d\x00%s\x00%d",
		p.Name, p.URLMustMatch, p.MaxDepth, p.URLMustNotMatch, p.MaxPagesPerHost,
	)
	sum := md5.Sum([]byte(raw))
	return yacymodel.Encode(sum[:])[:yacymodel.HashLength]
}
