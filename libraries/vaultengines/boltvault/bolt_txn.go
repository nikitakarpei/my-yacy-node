package boltvault

import (
	bolt "go.etcd.io/bbolt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type boltTxn struct {
	tx       *bolt.Tx
	writable bool
}

func (t boltTxn) Writable() bool { return t.writable }

func (t boltTxn) Bucket(name vault.Name) vault.EngineBucket {
	return boltBucket{
		name:    name,
		entries: t.tx.Bucket([]byte(name)),
		lengths: t.tx.Bucket([]byte(lengthBucket)),
	}
}
