// Package postingofferwait holds, for each posting short of replicas, how
// long it waits before the next offer, and grows that wait every time the
// posting misses redundancy again.
package postingofferwait

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const bucket vault.Name = "rwidistribution_offer_wait"

// Bounds are the shortest and longest waits between offers of one posting. A
// posting that has enough replicas waits the longest, so a posting
// that can never be placed costs no more than one that is already placed.
type Bounds struct {
	First   time.Duration
	Longest time.Duration
}

func (b Bounds) widened(previous time.Duration) time.Duration {
	return min(max(previous*2, b.First), b.Longest)
}

type Wait struct {
	waits *vault.Collection[time.Duration]
}

func Open(v *vault.Vault) (*Wait, error) {
	waits, err := vault.Register(v, bucket, waitCodec{})
	if err != nil {
		return nil, fmt.Errorf("register offer wait: %w", err)
	}

	return &Wait{waits: waits}, nil
}

// Widen doubles the posting's wait within the bounds and returns the wait now
// in force.
func (w *Wait) Widen(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
	bounds Bounds,
) (time.Duration, error) {
	key := postingschedule.PostingKey(word, url)
	previous, _, err := w.waits.Get(tx, key)
	if err != nil {
		return 0, fmt.Errorf("read offer wait: %w", err)
	}

	widened := bounds.widened(previous)
	if err := w.waits.Put(tx, key, widened); err != nil {
		return 0, fmt.Errorf("record offer wait: %w", err)
	}

	return widened, nil
}

// Forget returns the posting to the first wait.
func (w *Wait) Forget(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	if _, err := w.waits.Delete(tx, postingschedule.PostingKey(word, url)); err != nil {
		return fmt.Errorf("drop offer wait: %w", err)
	}

	return nil
}

func (w *Wait) PostingPurged(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) error {
	return w.Forget(tx, word, url)
}
