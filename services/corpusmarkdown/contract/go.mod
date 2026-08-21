module github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore

go 1.26

replace github.com/nikitakarpei/yacy-rwi-node/natstestserver => ../../../libraries/natstestserver

require github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0

replace github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract => ../../yacycrawler/contract

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../../../libraries/yacymodel

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../../libraries/canonicalurl
