module github.com/nikitakarpei/yacy-rwi-node/webresearchmcp

go 1.27

require (
	github.com/modelcontextprotocol/go-sdk v1.7.0
	github.com/nats-io/nats.go v1.52.0
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/natstestserver v0.0.0-00010101000000-000000000000
	github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/serviceruntime v0.0.0
	github.com/prometheus/client_golang v1.23.2
	google.golang.org/grpc v1.83.1
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/antithesishq/antithesis-sdk-go v0.7.0-default-no-op // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/jsonschema-go v0.4.3 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/minio/highwayhash v1.0.4 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.8.2 // indirect
	github.com/nats-io/nats-server/v2 v2.14.2 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nikitakarpei/yacy-rwi-node/pagefetch v0.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/oauth2 v0.36.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore => ../corpusmarkdown/contract

replace github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract => ../../services/pagescrape/contract

replace github.com/nikitakarpei/yacy-rwi-node/serviceruntime => ../../libraries/serviceruntime

replace github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract => ../yacycrawler/contract

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../../libraries/yacymodel

replace github.com/nikitakarpei/yacy-rwi-node/natstestserver => ../../libraries/natstestserver

replace github.com/nikitakarpei/yacy-rwi-node/pagefetch => ../../libraries/pagefetch
