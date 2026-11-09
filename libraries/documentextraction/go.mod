module github.com/nikitakarpei/yacy-rwi-node/documentextraction

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	golang.org/x/net v0.62.0
)

require golang.org/x/text v0.43.0 // indirect

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl
