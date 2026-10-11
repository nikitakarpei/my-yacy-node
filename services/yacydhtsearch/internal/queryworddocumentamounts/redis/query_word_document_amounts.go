// Package redis remembers on a Redis-protocol server, for the lifetime of an amount,
// the amount of documents in one partition for each query word, so that every service
// instance leads with the rarest word from the same amounts.
package redis

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type QueryWordDocumentAmountsObserver interface {
	DocumentAmountLookupFailed(ctx context.Context, word yacymodel.Hash, err error)
	DocumentAmountStoreFailed(ctx context.Context, word yacymodel.Hash, err error)
}

const keyPrefix = "query-word-document-amounts:"

type QueryWordDocumentAmounts struct {
	client   redis.UniversalClient
	lifetime time.Duration
	observer QueryWordDocumentAmountsObserver
}

func New(
	client redis.UniversalClient,
	lifetime time.Duration,
	observer QueryWordDocumentAmountsObserver,
) *QueryWordDocumentAmounts {
	return &QueryWordDocumentAmounts{client: client, lifetime: lifetime, observer: observer}
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
	stored, err := a.client.Get(ctx, keyOf(word)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false
	}
	if err != nil {
		a.observer.DocumentAmountLookupFailed(ctx, word, err)

		return 0, false
	}
	amount, err := strconv.Atoi(stored)
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
		if err := a.client.Set(ctx, keyOf(word), strconv.Itoa(amount), a.lifetime).
			Err(); err != nil {
			a.observer.DocumentAmountStoreFailed(ctx, word, err)
		}
	}
}

func keyOf(word yacymodel.Hash) string {
	return keyPrefix + word.String()
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
