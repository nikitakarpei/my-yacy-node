module github.com/nikitakarpei/yacy-rwi-node/documentextraction

go 1.26

require (
	github.com/nikitakarpei/yacy-rwi-node/pagelinks v0.0.0
	golang.org/x/net v0.55.0
)

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/pagelinks => ../../libraries/pagelinks
