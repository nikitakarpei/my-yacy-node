module github.com/nikitakarpei/yacy-rwi-node/corpustext

go 1.27.3

require (
	codeberg.org/readeck/go-readability/v2 v2.1.2 // indirect
	github.com/JohannesKaufmann/dom v0.3.1 // indirect
	github.com/JohannesKaufmann/html-to-markdown/v2 v2.5.2 // indirect
	github.com/RoaringBitmap/roaring/v2 v2.14.5 // indirect
	github.com/andybalholm/cascadia v1.3.4 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.7.0-default-no-op // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.24.2 // indirect
	github.com/blevesearch/bleve_index_api v1.4.1 // indirect
	github.com/blevesearch/geo v0.2.6 // indirect
	github.com/blevesearch/go-faiss v1.1.5 // indirect
	github.com/blevesearch/go-porterstemmer v1.0.3 // indirect
	github.com/blevesearch/gtreap v0.1.1 // indirect
	github.com/blevesearch/mmap-go v1.2.0 // indirect
	github.com/blevesearch/scorch_segment_api/v2 v2.4.10 // indirect
	github.com/blevesearch/segment v0.9.1 // indirect
	github.com/blevesearch/snowballstem v0.9.0 // indirect
	github.com/blevesearch/upsidedown_store_api v1.0.2 // indirect
	github.com/blevesearch/vellum v1.2.0 // indirect
	github.com/blevesearch/zapx/v11 v11.4.3 // indirect
	github.com/blevesearch/zapx/v12 v12.4.3 // indirect
	github.com/blevesearch/zapx/v13 v13.4.3 // indirect
	github.com/blevesearch/zapx/v14 v14.4.3 // indirect
	github.com/blevesearch/zapx/v15 v15.4.3 // indirect
	github.com/blevesearch/zapx/v16 v16.3.4 // indirect
	github.com/blevesearch/zapx/v17 v17.2.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudflare/circl v1.6.5 // indirect
	github.com/digitorus/pkcs7 v0.0.0-20260914070511-d678ea5ea03f // indirect
	github.com/digitorus/timestamp v0.0.0-20260914073129-b4b58b92aa51 // indirect
	github.com/go-shiori/dom v0.0.0-20230515143342-73569d674e1c // indirect
	github.com/gogs/chardet v0.0.0-20211120154057-b7413eaefb8f // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/itlightning/dateparse v0.2.1 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.20.2 // indirect
	github.com/minio/highwayhash v1.0.4 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/jwt/v2 v2.8.2 // indirect
	github.com/nats-io/nats-server/v2 v2.14.2 // indirect
	github.com/nats-io/nkeys v0.4.16 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nikitakarpei/hardware-attested-statements/verifier v0.3.0 // indirect
	github.com/nikitakarpei/yacy-rwi-node/attestedstatements v0.0.0-00010101000000-000000000000 // indirect
	github.com/nikitakarpei/yacy-rwi-node/optional v0.0.0 // indirect
	github.com/nikitakarpei/yacy-rwi-node/pagefetch v0.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/transparency-dev/merkle v0.0.2 // indirect
	go.etcd.io/bbolt v1.4.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/crypto v0.58.1-0.20261009181718-2f1f834330b2 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/searchdocument => ./contract

replace github.com/nikitakarpei/yacy-rwi-node/serviceruntime => ../../libraries/serviceruntime

replace github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract => ../yacycrawler/contract

replace github.com/nikitakarpei/yacy-rwi-node/yacymodel => ../../libraries/yacymodel

replace github.com/nikitakarpei/yacy-rwi-node/natstestserver => ../../libraries/natstestserver

replace github.com/nikitakarpei/yacy-rwi-node/pagefetch => ../../libraries/pagefetch

replace github.com/nikitakarpei/yacy-rwi-node/spamassessment => ../../libraries/spamassessment

replace github.com/nikitakarpei/yacy-rwi-node/pageformats => ../../libraries/pageformats

replace github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl

replace github.com/nikitakarpei/yacy-rwi-node/textsnippet => ../../libraries/textsnippet

require (
	github.com/blevesearch/bleve/v2 v2.6.1
	github.com/nats-io/nats.go v1.52.0
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/documentextraction v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/natstestserver v0.0.0-00010101000000-000000000000
	github.com/nikitakarpei/yacy-rwi-node/pageformats v0.0.0-00010101000000-000000000000
	github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/searchdocument v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/serviceruntime v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/textsnippet v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/wallclock v0.0.0
	github.com/prometheus/client_golang v1.23.2
)

replace github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract => ../../services/pagescrape/contract

replace github.com/nikitakarpei/yacy-rwi-node/documentextraction => ../../libraries/documentextraction

replace github.com/nikitakarpei/yacy-rwi-node/wallclock => ../../libraries/wallclock

replace github.com/nikitakarpei/yacy-rwi-node/optional => ../../libraries/optional

replace github.com/nikitakarpei/yacy-rwi-node/attestedstatements => ../../libraries/attestedstatements
