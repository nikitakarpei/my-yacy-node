package yacycrawlcontract

type CrawlPage struct {
	Crawls      []Crawl `json:"Crawls"`
	OlderCursor string  `json:"OlderCursor,omitzero"`
	NewerCursor string  `json:"NewerCursor,omitzero"`
}

type CrawlHost struct {
	Host                        string                        `json:"Host"`
	PendingPageVisits           int64                         `json:"PendingPageVisits"`
	CompletedPageVisitsByResult map[PageVisitResult]int64     `json:"CompletedPageVisitsByResult,omitzero"`
	DroppedPageVisitsByReason   map[PageVisitDropReason]int64 `json:"DroppedPageVisitsByReason,omitzero"`
}

type CrawlHostPage struct {
	Hosts          []CrawlHost `json:"Hosts"`
	NextCursor     string      `json:"NextCursor,omitzero"`
	PreviousCursor string      `json:"PreviousCursor,omitzero"`
}

type CrawlerStatus struct {
	Paused                bool  `json:"Paused"`
	PendingPageVisits     int64 `json:"PendingPageVisits"`
	AmountOfRunningCrawls int64 `json:"AmountOfRunningCrawls"`
}

type Pause struct {
	Paused bool `json:"Paused"`
}
