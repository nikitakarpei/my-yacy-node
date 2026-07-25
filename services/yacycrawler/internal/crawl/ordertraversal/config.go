package ordertraversal

import "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"

type Config struct {
	RunPageBudget    int
	VisitConcurrency int
	MaxAdmittedURLs  int
	Frontier         frontier.Config
}
