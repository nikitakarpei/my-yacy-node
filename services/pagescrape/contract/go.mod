module github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/pagefetch v0.0.0
)

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../../libraries/canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/pagefetch => ../../../libraries/pagefetch
