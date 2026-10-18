package pebblevault

import (
	"fmt"

	"github.com/cockroachdb/pebble/v2"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type memberSetChanges struct {
	writes        *pebble.Batch
	heldValues    heldValues
	presenceByKey map[string]memberPresence
}

func newMemberSetChanges(writes *pebble.Batch, heldValues heldValues) *memberSetChanges {
	return &memberSetChanges{
		writes:        writes,
		heldValues:    heldValues,
		presenceByKey: map[string]memberPresence{},
	}
}

func (c *memberSetChanges) mark(key []byte, member vault.StoredMember, isPresent bool) {
	presence, found := c.presenceByKey[string(key)]
	if !found {
		presence = memberPresence{}
		c.presenceByKey[string(key)] = presence
	}
	presence[member] = isPresent
}

func (c *memberSetChanges) appliedTo(
	key []byte,
	members []vault.StoredMember,
) []vault.StoredMember {
	presence := c.presenceByKey[string(key)]
	if len(presence) == 0 {
		return members
	}
	unchanged := make([]vault.StoredMember, 0, len(members))
	for _, member := range members {
		if _, changed := presence[member]; !changed {
			unchanged = append(unchanged, member)
		}
	}

	return append(unchanged, presence.changeOf().Added...)
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
	change := c.presenceByKey[key].changeOf()
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
