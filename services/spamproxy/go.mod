module github.com/nikitakarpei/yacy-rwi-node/spamproxy

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/serviceruntime v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/spamassessment v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/spammodel v0.0.0
	github.com/nikitakarpei/yacy-rwi-node/wallclock v0.0.0
	github.com/prometheus/client_golang v1.23.2
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	github.com/nikitakarpei/yacy-rwi-node/canonicalurl => ../../libraries/canonicalurl
	github.com/nikitakarpei/yacy-rwi-node/natstestserver => ../../libraries/natstestserver
	github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease => ../../libraries/processenvironmentlease
	github.com/nikitakarpei/yacy-rwi-node/serviceruntime => ../../libraries/serviceruntime
	github.com/nikitakarpei/yacy-rwi-node/spamassessment => ../../libraries/spamassessment
	github.com/nikitakarpei/yacy-rwi-node/spammodel => ../../libraries/spammodel
	github.com/nikitakarpei/yacy-rwi-node/wallclock => ../../libraries/wallclock
)
