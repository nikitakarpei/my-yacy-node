module github.com/nikitakarpei/yacy-rwi-node/vaultengines/boltvault

go 1.27

require (
	github.com/nikitakarpei/yacy-rwi-node/vault v0.0.0
	go.etcd.io/bbolt v1.4.3
)

require (
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
)

replace github.com/nikitakarpei/yacy-rwi-node/vault => ../../vault
