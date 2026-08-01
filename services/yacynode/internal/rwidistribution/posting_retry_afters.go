package rwidistribution

import (
	"time"
)

type postingRetryAfters struct {
	retryAfter map[postingIdentity]time.Duration
}

func newPostingRetryAfters() *postingRetryAfters {
	return &postingRetryAfters{
		retryAfter: make(map[postingIdentity]time.Duration),
	}
}

func (t *postingRetryAfters) Record(offer postingOffer, retryAfter time.Duration) {
	for _, posting := range offer.Postings {
		id := postingIdentity{Word: posting.WordHash, URL: posting.URLHash}
		if retryAfter > t.retryAfter[id] {
			t.retryAfter[id] = retryAfter
		}
	}
}

func (t *postingRetryAfters) RetryAfter(id postingIdentity) time.Duration {
	return t.retryAfter[id]
}
