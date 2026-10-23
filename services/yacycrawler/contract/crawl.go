package yacycrawlcontract

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type CrawlState string

const (
	CrawlStateRunning    CrawlState = "running"
	CrawlStatePaused     CrawlState = "paused"
	CrawlStateCancelling CrawlState = "cancelling"
	CrawlStateSucceeded  CrawlState = "succeeded"
	CrawlStateCancelled  CrawlState = "cancelled"
)

var CrawlStates = []CrawlState{
	CrawlStateRunning,
	CrawlStatePaused,
	CrawlStateCancelling,
	CrawlStateSucceeded,
	CrawlStateCancelled,
}

type Crawl struct {
	CrawlID                     string                        `json:"CrawlID"`
	OrderID                     string                        `json:"OrderID"`
	Label                       string                        `json:"Label,omitzero"`
	StartedAt                   time.Time                     `json:"StartedAt"`
	FinishedAt                  time.Time                     `json:"FinishedAt,omitzero"`
	State                       CrawlState                    `json:"State"`
	FirstSeedURL                canonicalurl.CanonicalURL     `json:"FirstSeedURL"`
	AmountOfSeedURLs            int64                         `json:"AmountOfSeedURLs"`
	PendingPageVisits           int64                         `json:"PendingPageVisits"`
	UnadmittedLinks             int64                         `json:"UnadmittedLinks"`
	Order                       *CrawlOrder                   `json:"Order,omitzero"`
	CompletedPageVisitsByResult map[PageVisitResult]int64     `json:"CompletedPageVisitsByResult,omitzero"`
	DroppedPageVisitsByReason   map[PageVisitDropReason]int64 `json:"DroppedPageVisitsByReason,omitzero"`
}
