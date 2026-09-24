// Package memory remembers in this process how many documents the peers hold
// for each query word, for as long as its lifetime allows and for as many words
// as its capacity allows.
package memory

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type QueryWordAmounts struct {
	amounts *expirable.LRU[yacymodel.Hash, int]
}

func New(capacity int, lifetime time.Duration) *QueryWordAmounts {
	return &QueryWordAmounts{
		amounts: expirable.NewLRU[yacymodel.Hash, int](capacity, nil, lifetime),
	}
}

func (a *QueryWordAmounts) AmountsOf(
	_ context.Context,
	words []yacymodel.Hash,
) map[yacymodel.Hash]int {
	amounts := make(map[yacymodel.Hash]int, len(words))
	for _, word := range words {
		amount, remembered := a.amounts.Get(word)
		if !remembered {
			continue
		}
		amounts[word] = amount
	}

	return amounts
}

func (a *QueryWordAmounts) Remember(_ context.Context, amounts map[yacymodel.Hash]int) {
	for word, amount := range amounts {
		a.amounts.Add(word, amount)
	}
}
