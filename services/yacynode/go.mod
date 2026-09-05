module github.com/nikitakarpei/yacy-rwi-node/yacynode

go 1.27

require (
	codeberg.org/readeck/go-readability/v2 v2.1.2 // indirect
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/JohannesKaufmann/dom v0.3.1 // indirect
	github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.2 // indirect
	github.com/RaduBerinde/axisds v0.1.0 // indirect
	github.com/RaduBerinde/btreemap v0.0.0-20250419174037-3d62b7205d54 // indirect
	github.com/andybalholm/cascadia v1.3.4 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.7.0-default-no-op // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/crlib v0.0.0-20241112164430-1264a2edc35b // indirect
	github.com/cockroachdb/errors v1.11.3 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/pebble/v2 v2.1.7 // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/swiss v0.0.0-20260820225851-333444432258 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/getsentry/sentry-go v0.27.0 // indirect
	github.com/go-shiori/dom v0.0.0-20230515143342-73569d674e1c // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/golang/snappy v0.0.5-0.20231225225746-43d5d4cd4e0e // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/itlightning/dateparse v0.2.1 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/minio/highwayhash v1.0.4 // indirect
	github.com/minio/minlz v1.0.1-0.20250507153514-87eb42fe8882 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.8.2 // indirect
	github.com/nats-io/nats-server/v2 v2.14.2 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nikitakarpei/yacy-rwi-node/pagefetch v0.0.0 // indirect
	github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease v0.0.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rogpeppe/go-internal v1.11.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/exp v0.0.0-20230626212559-97b1e661b5df // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/searchdocument => ../corpustext/contract

replace github.com/nikitakarpei/yacy-rwi-node/serviceruntime => ../../libraries/serviceruntime

replace github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract => ../yacycrawler/contract

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../../libraries/yacymodel

replace github.com/nikitakarpei/yacy-rwi-node/yacyproto => ../../libraries/yacyproto

replace github.com/nikitakarpei/yacy-rwi-node/natstestserver => ../../libraries/natstestserver

replace github.com/nikitakarpei/yacy-rwi-node/storedfields => ../../libraries/storedfields

replace github.com/nikitakarpei/yacy-rwi-node/vault => ../../libraries/vault

replace github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault => ../../libraries/vaultengines/pebblevault

replace github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault => ../../libraries/vaultengines/memoryvault

replace github.com/nikitakarpei/yacy-rwi-node/pagefetch => ../../libraries/pagefetch

replace github.com/nikitakarpei/yacy-rwi-node/pageformats => ../../libraries/pageformats

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl

require (
	github.com/nats-io/nats.go v1.52.0
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/documentextraction v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/natstestserver v0.0.0-00010101000000-000000000000
	github.com/nikitakarpei/yacy-rwi-node/pageformats v0.0.0-00010101000000-000000000000
	github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/serviceruntime v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/storedfields v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/vault v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/vaultengines/pebblevault v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/yacymodel v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/yacyproto v0.0.0
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
)

replace github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract => ../../services/pagescrape/contract

replace github.com/nikitakarpei/yacy-rwi-node/documentextraction => ../../libraries/documentextraction

replace github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease => ../../libraries/processenvironmentlease
