module github.com/nikitakarpei/yacy-rwi-node/yacyproto

go 1.27

require github.com/nikitakarpei/yacy-rwi-node/yacymodel v0.0.0

require (
	github.com/cloudflare/circl v1.6.5 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../yacymodel
