module github.com/nikitakarpei/yacy-rwi-node/pagelinks

go 1.26

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	golang.org/x/net v0.55.0
)

require golang.org/x/text v0.37.0 // indirect

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl
