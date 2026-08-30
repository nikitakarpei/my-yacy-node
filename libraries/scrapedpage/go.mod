module github.com/nikitakarpei/yacy-rwi-node/scrapedpage

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/pagefetch v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract v0.0.0
)

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/pagefetch => ../pagefetch

replace github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract => ../scraperequestcontract
