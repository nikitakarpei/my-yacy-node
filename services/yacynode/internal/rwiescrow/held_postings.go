package rwiescrow

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type HeldPostings struct {
	vault    *vault.Vault
	held     *vault.Collection[heldPosting]
	expiry   *vault.Set
	admitter rwipostings.PostingAdmitter
	observer HoldObserver
	capacity int
	now      func() time.Time
}

func (h *HeldPostings) Hold(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	identity := postingIdentity{Word: posting.WordHash, URL: posting.URLHash}
	heldAt := h.now()

	previous, found, err := h.held.Get(tx, heldKey(identity))
	if err != nil {
		return fmt.Errorf("read held posting: %w", err)
	}
	if found {
		if _, err := h.expiry.Remove(tx, expiryKey(previous.HeldAt, identity)); err != nil {
			return fmt.Errorf("drop stale posting hold time: %w", err)
		}
	} else {
		full, err := h.atCapacity(tx)
		if err != nil {
			return err
		}
		if full {
			h.observer.ObserveRefused(1)

			return nil
		}
	}

	if err := h.held.Put(tx, heldKey(identity), heldPosting{
		HeldAt:  heldAt,
		Posting: posting,
	}); err != nil {
		return fmt.Errorf("hold posting: %w", err)
	}
	if err := h.expiry.Add(tx, expiryKey(heldAt, identity)); err != nil {
		return fmt.Errorf("record posting hold time: %w", err)
	}
	if !found {
		h.observer.ObserveHeld(1)
	}

	return nil
}

func (h *HeldPostings) atCapacity(tx *vault.Txn) (bool, error) {
	if h.capacity <= 0 {
		return false, nil
	}
	length, err := h.held.Len(tx)
	if err != nil {
		return false, fmt.Errorf("read held posting length: %w", err)
	}

	return length >= h.capacity, nil
}

func (h *HeldPostings) URLStored(
	tx *vault.Txn,
	hash yacymodel.URLHash,
	_ yacymodel.Optional[yacymodel.CalendarDay],
) error {
	waiting, err := h.postingsWaitingFor(tx, hash)
	if err != nil {
		return err
	}
	if len(waiting) == 0 {
		return nil
	}

	for _, held := range waiting {
		if err := h.release(tx, held); err != nil {
			return err
		}
	}
	h.observer.ObserveReleased(len(waiting))

	return nil
}

func (h *HeldPostings) URLPurged(*vault.Txn, yacymodel.URLHash) error {
	return nil
}

func (h *HeldPostings) Expire(
	ctx context.Context,
	holdFor time.Duration,
	limit int,
) (int, error) {
	if limit <= 0 {
		return 0, nil
	}

	cutoff := heldAtPrefix(h.now().Add(-holdFor))
	var expired int
	err := h.vault.Update(ctx, func(tx *vault.Txn) error {
		keys, err := h.expiryKeysBefore(tx, cutoff, limit)
		if err != nil {
			return err
		}
		for _, key := range keys {
			identity, err := parseExpiryKey(key)
			if err != nil {
				return err
			}
			if err := h.drop(tx, key, identity); err != nil {
				return err
			}
		}
		expired = len(keys)

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("expire held postings: %w", err)
	}

	return expired, nil
}

func (h *HeldPostings) Count(ctx context.Context) (int, error) {
	var count int
	err := h.vault.View(ctx, func(tx *vault.Txn) error {
		length, err := h.held.Len(tx)
		if err != nil {
			return fmt.Errorf("read held posting length: %w", err)
		}
		count = length

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("held posting count: %w", err)
	}

	return count, nil
}

func (h *HeldPostings) Capacity() int {
	return h.capacity
}

func (h *HeldPostings) postingsWaitingFor(
	tx *vault.Txn,
	hash yacymodel.URLHash,
) ([]heldPosting, error) {
	var waiting []heldPosting
	err := h.held.Scan(
		tx,
		vault.Key(hash.String()),
		func(key vault.Key, held heldPosting) (bool, error) {
			identity, err := parseHeldKey(key)
			if err != nil {
				return false, err
			}
			held.Posting.WordHash = identity.Word
			waiting = append(waiting, held)

			return true, nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read postings waiting for url: %w", err)
	}

	return waiting, nil
}

func (h *HeldPostings) expiryKeysBefore(
	tx *vault.Txn,
	cutoff vault.Key,
	limit int,
) ([]vault.Key, error) {
	keys := make([]vault.Key, 0, limit)
	err := h.expiry.Scan(tx, nil, func(key vault.Key) (bool, error) {
		if len(key) < heldAtDigits || bytes.Compare(key[:heldAtDigits], cutoff) >= 0 {
			return false, nil
		}
		keys = append(keys, key)

		return len(keys) < limit, nil
	})
	if err != nil {
		return nil, fmt.Errorf("read expired posting holds: %w", err)
	}

	return keys, nil
}

func (h *HeldPostings) release(tx *vault.Txn, held heldPosting) error {
	identity := postingIdentity{Word: held.Posting.WordHash, URL: held.Posting.URLHash}
	if err := h.admitter.Admit(tx, held.Posting); err != nil {
		return fmt.Errorf("admit released posting: %w", err)
	}

	return h.drop(tx, expiryKey(held.HeldAt, identity), identity)
}

func (h *HeldPostings) drop(
	tx *vault.Txn,
	expiryRow vault.Key,
	identity postingIdentity,
) error {
	if _, err := h.held.Delete(tx, heldKey(identity)); err != nil {
		return fmt.Errorf("drop held posting: %w", err)
	}
	if _, err := h.expiry.Remove(tx, expiryRow); err != nil {
		return fmt.Errorf("drop posting hold time: %w", err)
	}

	return nil
}
