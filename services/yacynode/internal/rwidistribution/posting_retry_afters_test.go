package rwidistribution

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingOfferTallyKeepsLongestRetryAfter(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	id := postingIdentity{Word: word, URL: url}
	posting := fakePosting(word, url)
	tally := newPostingRetryAfters()

	tally.Record(
		postingOffer{
			Peer:     seed(yacymodel.WordHash("a")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		time.Minute,
	)
	tally.Record(
		postingOffer{
			Peer:     seed(yacymodel.WordHash("b")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		5*time.Minute,
	)

	if got := tally.RetryAfter(id); got != 5*time.Minute {
		t.Fatalf("RetryAfter = %v, want 5m", got)
	}
}

func TestPostingOfferTallyLeavesUnrelatedPostingRetryAfterUntouched(t *testing.T) {
	word1, url1 := yacymodel.WordHash("w1"), urlHash("u1")
	word2, url2 := yacymodel.WordHash("w2"), urlHash("u2")
	id2 := postingIdentity{Word: word2, URL: url2}
	offer := postingOffer{
		Peer: seed(yacymodel.WordHash("a")),
		Postings: []yacymodel.RWIPosting{
			fakePosting(word1, url1),
		},
	}
	tally := newPostingRetryAfters()

	tally.Record(offer, time.Minute)

	if got := tally.RetryAfter(id2); got != 0 {
		t.Fatalf("RetryAfter = %v, want 0 for an unrelated posting", got)
	}
}
