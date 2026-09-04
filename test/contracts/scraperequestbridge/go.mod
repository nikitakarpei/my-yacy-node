module github.com/nikitakarpei/yacy-rwi-node/test/contracts/scraperequestbridge

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract v0.0.0
)

require github.com/nikitakarpei/yacy-rwi-node/pagefetch v0.0.0 // indirect

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../../libraries/canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../../../libraries/yacymodel

replace github.com/nikitakarpei/yacy-rwi-node/pagefetch => ../../../libraries/pagefetch

replace github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract => ../../../services/pagescrape/contract

replace github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract => ../../../services/yacycrawler/contract
