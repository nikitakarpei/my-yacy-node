package memoryvault

import (
	"sort"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

type memAmounts struct {
	entries map[string]int
}

func (a memAmounts) Get(key []byte) (int, error) {
	return a.entries[string(key)], nil
}

func (a memAmounts) Raise(key []byte, by uint) error {
	a.changeBy(key, int(by))

	return nil
}

func (a memAmounts) changeBy(key []byte, change int) {
	changed := a.entries[string(key)] + change
	if changed <= 0 {
		delete(a.entries, string(key))

		return
	}
	a.entries[string(key)] = changed
}

func (a memAmounts) Lower(key []byte, by uint) error {
	a.changeBy(key, -int(by))

	return nil
}

func (a memAmounts) Scan(keys vault.KeyRange, fn func(key []byte, amount int) (bool, error)) error {
	for _, key := range orderedAmountKeysOf(a.entries, keys) {
		keep, err := fn([]byte(key), a.entries[key])
		if err != nil || !keep {
			return err
		}
	}

	return nil
}

func orderedAmountKeysOf(entries map[string]int, keys vault.KeyRange) []string {
	firstIncluded, firstExcluded := keys.Bounds()

	ordered := make([]string, 0, len(entries))
	for key := range entries {
		if isWithinBounds(key, firstIncluded, firstExcluded) {
			ordered = append(ordered, key)
		}
	}
	sort.Strings(ordered)

	return ordered
}
