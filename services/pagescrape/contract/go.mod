module github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract

go 1.27.3

require (
	github.com/nikitakarpei/yacy-rwi-node/attestedstatements v0.0.0-00010101000000-000000000000
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/intotostatements v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/pagefetch v0.0.0
)

require (
	github.com/cloudflare/circl v1.6.5 // indirect
	github.com/digitorus/pkcs7 v0.0.0-20260914070511-d678ea5ea03f // indirect
	github.com/digitorus/timestamp v0.0.0-20260914073129-b4b58b92aa51 // indirect
	github.com/nikitakarpei/hardware-attested-statements/verifier v0.3.0 // indirect
	github.com/nikitakarpei/yacy-rwi-node/optional v0.0.0 // indirect
	github.com/transparency-dev/merkle v0.0.2 // indirect
	golang.org/x/crypto v0.58.1-0.20261009181718-2f1f834330b2 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../../libraries/canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/pagefetch => ../../../libraries/pagefetch

replace github.com/nikitakarpei/yacy-rwi-node/spamassessment => ../../../libraries/spamassessment

replace github.com/nikitakarpei/yacy-rwi-node/optional => ../../../libraries/optional

replace github.com/nikitakarpei/yacy-rwi-node/intotostatements => ../../../libraries/intotostatements

replace github.com/nikitakarpei/yacy-rwi-node/attestedstatements => ../../../libraries/attestedstatements
