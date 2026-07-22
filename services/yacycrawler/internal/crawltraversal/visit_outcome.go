package crawltraversal

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlfrontier"
)

type visitOutcome struct {
	entry      crawlfrontier.Entry
	candidates []discoveredLink
	counted    bool
	deferred   bool
	deferFor   time.Duration
	transient  bool
	err        error
}

type discoveredLink struct {
	url   string
	depth int
}

func discoveredLinks(urls []string, depth int) []discoveredLink {
	links := make([]discoveredLink, 0, len(urls))
	for _, url := range urls {
		links = append(links, discoveredLink{url: url, depth: depth})
	}
	return links
}
