package pebblevault

import (
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type memberSetChanges struct {
	writes        *pebble.Batch
	heldValues    heldValues
	presenceByKey map[string]map[vault.StoredMember]bool
}

func newMemberSetChanges(writes *pebble.Batch, heldValues heldValues) *memberSetChanges {
	return &memberSetChanges{
		writes:        writes,
		heldValues:    heldValues,
		presenceByKey: map[string]map[vault.StoredMember]bool{},
	}
}

func (c *memberSetChanges) mark(key []byte, member vault.StoredMember, isPresent bool) {
	presenceByMember, found := c.presenceByKey[string(key)]
	if !found {
		presenceByMember = map[vault.StoredMember]bool{}
		c.presenceByKey[string(key)] = presenceByMember
	}
	presenceByMember[member] = isPresent
}

func (c *memberSetChanges) appliedTo(
	key []byte,
	members []vault.StoredMember,
) []vault.StoredMember {
	presenceByMember := c.presenceByKey[string(key)]
	if len(presenceByMember) == 0 {
		return members
	}
	applied := make([]vault.StoredMember, 0, len(members)+len(presenceByMember))
	for _, member := range members {
		if _, changed := presenceByMember[member]; !changed {
			applied = append(applied, member)
		}
	}
	for member, isPresent := range presenceByMember {
		if isPresent {
			applied = append(applied, member)
		}
	}

	return applied
}

func (c *memberSetChanges) drop(key []byte) {
	delete(c.presenceByKey, string(key))
}

func (c *memberSetChanges) stageWithin(firstIncluded, firstExcluded []byte) error {
	for key := range c.presenceByKey {
		if key < string(firstIncluded) || key >= string(firstExcluded) {
			continue
		}
		if err := c.stage(key); err != nil {
			return err
		}
	}

	return nil
}

func (c *memberSetChanges) stage(key string) error {
	var change vault.MemberSetChange
	for member, isPresent := range c.presenceByKey[key] {
		if isPresent {
			change.Added = append(change.Added, member)
		} else {
			change.Removed = append(change.Removed, member)
		}
	}
	delete(c.presenceByKey, key)
	if len(change.Added)+len(change.Removed) == 0 {
		return nil
	}
	if err := c.writes.Merge([]byte(key), change.Encoded(), pebble.NoSync); err != nil {
		return fmt.Errorf("store member set change: %w", err)
	}
	c.heldValues.forget([]byte(key))

	return nil
}

func (c *memberSetChanges) stageAll() error {
	for key := range c.presenceByKey {
		if err := c.stage(key); err != nil {
			return err
		}
	}

	return nil
}
