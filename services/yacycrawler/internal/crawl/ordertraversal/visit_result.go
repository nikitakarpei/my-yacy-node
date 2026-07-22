package ordertraversal

import "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"

type pageVisitResult struct {
	entry   entry
	outcome pagevisit.VisitOutcome
	err     error
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
