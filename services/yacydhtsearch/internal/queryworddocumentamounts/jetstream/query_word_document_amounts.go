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

type QueryWordDocumentAmountsObserver interface {
	DocumentAmountLookupFailed(ctx context.Context, word yacymodel.Hash, err error)
	DocumentAmountStoreFailed(ctx context.Context, word yacymodel.Hash, err error)
}

type QueryWordDocumentAmounts struct {
	bucket   natsjetstream.KeyValue
	observer QueryWordDocumentAmountsObserver
}

func New(
	bucket natsjetstream.KeyValue,
	observer QueryWordDocumentAmountsObserver,
) *QueryWordDocumentAmounts {
	return &QueryWordDocumentAmounts{bucket: bucket, observer: observer}
}

func (a *QueryWordDocumentAmounts) DocumentAmountsOf(
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

func (a *QueryWordDocumentAmounts) amountOf(ctx context.Context, word yacymodel.Hash) (int, bool) {
	entry, err := a.bucket.Get(ctx, word.String())
	if errors.Is(err, natsjetstream.ErrKeyNotFound) {
		return 0, false
	}
	if err != nil {
		a.observer.DocumentAmountLookupFailed(ctx, word, err)

		return 0, false
	}
	amount, err := strconv.Atoi(string(entry.Value()))
	if err != nil {
		a.observer.DocumentAmountLookupFailed(ctx, word, err)

		return 0, false
	}

	return amount, true
}

func (a *QueryWordDocumentAmounts) Remember(
	ctx context.Context,
	documentAmounts map[yacymodel.Hash]int,
) {
	for word, amount := range documentAmounts {
		if _, err := a.bucket.Put(ctx, word.String(), []byte(strconv.Itoa(amount))); err != nil {
			a.observer.DocumentAmountStoreFailed(ctx, word, err)
		}
	}
}

type QueryWordDocumentAmountsObservers []QueryWordDocumentAmountsObserver

func (observers QueryWordDocumentAmountsObservers) DocumentAmountLookupFailed(
	ctx context.Context,
	word yacymodel.Hash,
	err error,
) {
	for _, observer := range observers {
		observer.DocumentAmountLookupFailed(ctx, word, err)
	}
}

func (observers QueryWordDocumentAmountsObservers) DocumentAmountStoreFailed(
	ctx context.Context,
	word yacymodel.Hash,
	err error,
) {
	for _, observer := range observers {
		observer.DocumentAmountStoreFailed(ctx, word, err)
	}
}
