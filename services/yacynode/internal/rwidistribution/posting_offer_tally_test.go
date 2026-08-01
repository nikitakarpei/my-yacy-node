package rwidistribution

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPostingOfferTallyCountsAcceptedCopiesAcrossPeers(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	id := postingIdentity{Word: word, URL: url}
	posting := fakePosting(word, url)
	tally := newPostingOfferTally()

	tally.RecordOffer(
		postingOffer{
			Peer:     seed(yacymodel.WordHash("a")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		postingOfferReceipt{Outcome: postingOfferAccepted},
	)
	tally.RecordOffer(
		postingOffer{
			Peer:     seed(yacymodel.WordHash("b")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		postingOfferReceipt{Outcome: postingOfferAccepted},
	)

	if got := tally.AcceptedCopies(id); got != 2 {
		t.Fatalf("AcceptedCopies = %v, want 2", got)
	}
}

func TestPostingOfferTallyKeepsLongestRetryAfter(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	id := postingIdentity{Word: word, URL: url}
	posting := fakePosting(word, url)
	tally := newPostingOfferTally()

	tally.RecordOffer(
		postingOffer{
			Peer:     seed(yacymodel.WordHash("a")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		postingOfferReceipt{Outcome: postingOfferDeferred, RetryAfter: time.Minute},
	)
	tally.RecordOffer(
		postingOffer{
			Peer:     seed(yacymodel.WordHash("b")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		postingOfferReceipt{Outcome: postingOfferDeferred, RetryAfter: 5 * time.Minute},
	)

	if got := tally.RetryAfter(id); got != 5*time.Minute {
		t.Fatalf("RetryAfter = %v, want 5m", got)
	}
}

func TestPostingOfferTallyExcludesPostingWithoutMetadataFromAcceptedCopies(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	id := postingIdentity{Word: word, URL: url}
	posting := fakePosting(word, url)
	offer := postingOffer{
		Peer:     seed(yacymodel.WordHash("a")),
		Postings: []yacymodel.RWIPosting{posting},
	}
	tally := newPostingOfferTally()

	tally.RecordOffer(offer, postingOfferReceipt{Outcome: postingOfferAccepted})
	tally.RecordURLsWithoutMetadata(offer, []yacymodel.URLHash{url})

	if got := tally.AcceptedCopies(id); got != 0 {
		t.Fatalf("AcceptedCopies = %v, want 0 once metadata delivery fails", got)
	}
}

func TestPostingOfferTallyLeavesUnrelatedPostingAcceptedCopiesUntouched(t *testing.T) {
	word1, url1 := yacymodel.WordHash("w1"), urlHash("u1")
	word2, url2 := yacymodel.WordHash("w2"), urlHash("u2")
	id2 := postingIdentity{Word: word2, URL: url2}
	offer := postingOffer{
		Peer: seed(yacymodel.WordHash("a")),
		Postings: []yacymodel.RWIPosting{
			fakePosting(word1, url1),
			fakePosting(word2, url2),
		},
	}
	tally := newPostingOfferTally()

	tally.RecordOffer(offer, postingOfferReceipt{Outcome: postingOfferAccepted})
	tally.RecordURLsWithoutMetadata(offer, []yacymodel.URLHash{url1})

	if got := tally.AcceptedCopies(id2); got != 1 {
		t.Fatalf("AcceptedCopies = %v, want 1 for the posting whose metadata reached the peer", got)
	}
}
