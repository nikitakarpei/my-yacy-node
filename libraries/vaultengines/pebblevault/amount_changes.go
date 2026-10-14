package pebblevault

import (
	"fmt"

	"github.com/cockroachdb/pebble/v2"
)

type amountChanges struct {
	writes     *pebble.Batch
	heldValues heldValues
	sumByKey   map[string]int64
}

func newAmountChanges(writes *pebble.Batch, heldValues heldValues) *amountChanges {
	return &amountChanges{writes: writes, heldValues: heldValues, sumByKey: map[string]int64{}}
}

func (c *amountChanges) add(key []byte, change int64) {
	c.sumByKey[string(key)] += change
}

func (c *amountChanges) changeAt(key []byte) int64 {
	return c.sumByKey[string(key)]
}

func (c *amountChanges) stageWithin(firstIncluded, firstExcluded []byte) error {
	for key, sum := range c.sumByKey {
		if key < string(firstIncluded) || key >= string(firstExcluded) {
			continue
		}
		if err := c.stage(key, sum); err != nil {
			return err
		}
	}

	return nil
}

func (c *amountChanges) stage(key string, sum int64) error {
	delete(c.sumByKey, key)
	if sum == 0 {
		return nil
	}
	if err := c.writes.Merge([]byte(key), encodedAmount(sum), pebble.NoSync); err != nil {
		return fmt.Errorf("store amount: %w", err)
	}
	c.heldValues.forget([]byte(key))

	return nil
}

func (c *amountChanges) stageAll() error {
	for key, sum := range c.sumByKey {
		if err := c.stage(key, sum); err != nil {
			return err
		}
	}

	return nil
}
