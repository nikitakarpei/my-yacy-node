package searchpages

import "github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"

type sourceText struct {
	meaning string
	failure string
}

var sourceTexts = map[searchqueries.Source]sourceText{
	searchqueries.CrawledPages: {
		meaning: "only your crawled pages",
		failure: "Failed to search your crawled pages.",
	},
	searchqueries.YaCyNetwork: {
		meaning: "only the YaCy network",
		failure: "Failed to search the YaCy network.",
	},
}
