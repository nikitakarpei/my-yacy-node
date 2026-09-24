// Package jetstream remembers in a NATS key-value bucket how many documents
// the peers hold for each query word, so that every service instance leads
// with the rarest word from the same amounts.
package jetstream

import (
	"context"
	"errors"
	"strconv"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type QueryWordAmountsObserver interface {
	AmountLookupFailed(ctx context.Context, word yacymodel.Hash, err error)
	AmountStoreFailed(ctx context.Context, word yacymodel.Hash, err error)
}

type QueryWordAmounts struct {
	bucket   natsjetstream.KeyValue
	observer QueryWordAmountsObserver
}

func New(bucket natsjetstream.KeyValue, observer QueryWordAmountsObserver) *QueryWordAmounts {
	return &QueryWordAmounts{bucket: bucket, observer: observer}
}

func (a *QueryWordAmounts) AmountsOf(
	ctx context.Context,
	words []yacymodel.Hash,
) map[yacymodel.Hash]int {
	amounts := make(map[yacymodel.Hash]int, len(words))
	for _, word := range words {
		amount, remembered := a.amountOf(ctx, word)
		if !remembered {
			continue
		}
		amounts[word] = amount
	}

	return amounts
}

func (a *QueryWordAmounts) amountOf(ctx context.Context, word yacymodel.Hash) (int, bool) {
	entry, err := a.bucket.Get(ctx, word.String())
	if errors.Is(err, natsjetstream.ErrKeyNotFound) {
		return 0, false
	}
	if err != nil {
		a.observer.AmountLookupFailed(ctx, word, err)

		return 0, false
	}
	amount, err := strconv.Atoi(string(entry.Value()))
	if err != nil {
		a.observer.AmountLookupFailed(ctx, word, err)

		return 0, false
	}

	return amount, true
}

func (a *QueryWordAmounts) Remember(ctx context.Context, amounts map[yacymodel.Hash]int) {
	for word, amount := range amounts {
		if _, err := a.bucket.Put(ctx, word.String(), []byte(strconv.Itoa(amount))); err != nil {
			a.observer.AmountStoreFailed(ctx, word, err)
		}
	}
}

type QueryWordAmountsObservers []QueryWordAmountsObserver

func (observers QueryWordAmountsObservers) AmountLookupFailed(
	ctx context.Context,
	word yacymodel.Hash,
	err error,
) {
	for _, observer := range observers {
		observer.AmountLookupFailed(ctx, word, err)
	}
}

func (observers QueryWordAmountsObservers) AmountStoreFailed(
	ctx context.Context,
	word yacymodel.Hash,
	err error,
) {
	for _, observer := range observers {
		observer.AmountStoreFailed(ctx, word, err)
	}
}
