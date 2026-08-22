module github.com/nikitakarpei/yacy-rwi-node/yacynode

go 1.26

require (
	github.com/nats-io/nats.go v1.52.0
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/pagescrape v0.0.0-00010101000000-000000000000
	github.com/nikitakarpei/yacy-rwi-node/serviceruntime v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/vault v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/vaultengines/boltvault v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/yacymodel v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/yacyproto v0.0.0
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
)

require (
	codeberg.org/readeck/go-readability/v2 v2.1.2 // indirect
	github.com/JohannesKaufmann/dom v0.3.1 // indirect
	github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.2 // indirect
	github.com/andybalholm/cascadia v1.3.4 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-shiori/dom v0.0.0-20230515143342-73569d674e1c // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/itlightning/dateparse v0.2.1 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	go.etcd.io/bbolt v1.4.3 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/searchdocument => ../corpustext/contract

replace github.com/nikitakarpei/yacy-rwi-node/serviceruntime => ../../libraries/serviceruntime

replace github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract => ../yacycrawler/contract

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../../libraries/yacymodel

replace github.com/nikitakarpei/yacy-rwi-node/yacyproto => ../../libraries/yacyproto

replace github.com/nikitakarpei/yacy-rwi-node/natstestserver => ../../libraries/natstestserver

replace github.com/nikitakarpei/yacy-rwi-node/vault => ../../libraries/vault

replace github.com/nikitakarpei/yacy-rwi-node/vaultengines/boltvault => ../../libraries/vaultengines/boltvault

replace github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault => ../../libraries/vaultengines/memoryvault

replace github.com/nikitakarpei/yacy-rwi-node/pagescrape => ../../libraries/pagescrape

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl

require (
	github.com/nikitakarpei/yacy-rwi-node/documentextraction v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract v0.0.0
)

replace github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract => ../../libraries/scraperequestcontract

replace github.com/nikitakarpei/yacy-rwi-node/documentextraction => ../../libraries/documentextraction
