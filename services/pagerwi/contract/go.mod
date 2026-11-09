module github.com/nikitakarpei/yacy-rwi-node/pagerwicontract

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/yacymodel v0.0.0
)

require (
	github.com/cloudflare/circl v1.6.5 // indirect
	golang.org/x/crypto v0.58.1-0.20261009181718-2f1f834330b2 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../../libraries/canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../../../libraries/yacymodel
