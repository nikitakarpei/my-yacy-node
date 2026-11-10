module github.com/nikitakarpei/yacy-rwi-node/pageattestationcheck

go 1.27.3

require (
	github.com/digitorus/timestamp v0.0.0-20260914073129-b4b58b92aa51
	github.com/nikitakarpei/hardware-attested-statements/verifier v0.3.0
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/pagerwicontract v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/yacymodel v0.0.0
	github.com/transparency-dev/merkle v0.0.2
)

require (
	github.com/cloudflare/circl v1.6.5 // indirect
	github.com/digitorus/pkcs7 v0.0.0-20260914070511-d678ea5ea03f // indirect
	github.com/nikitakarpei/yacy-rwi-node/intotostatements v0.0.0 // indirect
	github.com/nikitakarpei/yacy-rwi-node/storedfields v0.0.0 // indirect
	golang.org/x/crypto v0.58.1-0.20261009181718-2f1f834330b2 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/pagerwicontract => ../../services/pagerwi/contract

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../yacymodel

replace github.com/nikitakarpei/yacy-rwi-node/storedfields => ../storedfields

replace github.com/nikitakarpei/yacy-rwi-node/intotostatements => ../intotostatements
