package boltvault

import (
	bolt "go.etcd.io/bbolt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/memberset"
)

type boltTxn struct {
	tx       *bolt.Tx
	writable bool
}

func (t boltTxn) Writable() bool { return t.writable }

func (t boltTxn) Records(name vault.Name) vault.EngineRecords {
	return boltRecords{
		name:    name,
		entries: t.tx.Bucket([]byte(name)),
		lengths: t.tx.Bucket([]byte(lengthBucket)),
	}
}

func (t boltTxn) MemberSets(name vault.Name) vault.EngineMemberSets {
	return memberset.OverRecords(t.Records(name))
}

func (t boltTxn) Amounts(name vault.Name) vault.EngineAmounts {
	return boltAmounts{entries: t.tx.Bucket([]byte(name))}
}
