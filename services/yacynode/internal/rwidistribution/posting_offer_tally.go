package rwidistribution

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingOfferTally struct {
	accepted        map[postingIdentity]int
	retryAfter      map[postingIdentity]time.Duration
	withoutMetadata map[postingIdentity]struct{}
}

func newPostingOfferTally() *postingOfferTally {
	return &postingOfferTally{
		accepted:        make(map[postingIdentity]int),
		retryAfter:      make(map[postingIdentity]time.Duration),
		withoutMetadata: make(map[postingIdentity]struct{}),
	}
}

func (t *postingOfferTally) RecordOffer(offer postingOffer, receipt postingOfferReceipt) {
	for _, posting := range offer.Postings {
		id := postingIdentity{Word: posting.WordHash, URL: posting.URLHash}
		if receipt.Outcome == postingOfferAccepted {
			t.accepted[id]++
		}
		if receipt.RetryAfter > t.retryAfter[id] {
			t.retryAfter[id] = receipt.RetryAfter
		}
	}
}

func (t *postingOfferTally) RecordURLsWithoutMetadata(
	offer postingOffer,
	urls []yacymodel.URLHash,
) {
	if len(urls) == 0 {
		return
	}

	missing := make(map[yacymodel.URLHash]struct{}, len(urls))
	for _, url := range urls {
		missing[url] = struct{}{}
	}

	for _, posting := range offer.Postings {
		if _, without := missing[posting.URLHash]; !without {
			continue
		}

		id := postingIdentity{Word: posting.WordHash, URL: posting.URLHash}
		t.withoutMetadata[id] = struct{}{}
		if t.accepted[id] > 0 {
			t.accepted[id]--
		}
	}
}

func (t *postingOfferTally) AcceptedCopies(id postingIdentity) int {
	return t.accepted[id]
}

func (t *postingOfferTally) RetryAfter(id postingIdentity) time.Duration {
	return t.retryAfter[id]
}
