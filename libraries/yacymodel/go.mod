module github.com/nikitakarpei/yacy-rwi-node/yacymodel

go 1.27

require (
	github.com/cloudflare/circl v1.6.5
	github.com/nikitakarpei/yacy-rwi-node/storedfields v0.0.0
)

require (
	golang.org/x/crypto v0.58.1-0.20261009181718-2f1f834330b2 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/storedfields => ../storedfields
