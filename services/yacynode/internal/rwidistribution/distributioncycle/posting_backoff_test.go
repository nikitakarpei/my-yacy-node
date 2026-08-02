package distributioncycle

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
)

func TestPostingBackoffKeepsLongestBackoff(t *testing.T) {
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	identity := postingschedule.Identity{Word: word, URL: url}
	posting := fakePosting(word, url)
	backoff := newPostingBackoff()

	backoff.Record(
		offer{
			Peer:     seed(yacymodel.WordHash("a")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		time.Minute,
	)
	backoff.Record(
		offer{
			Peer:     seed(yacymodel.WordHash("b")),
			Postings: []yacymodel.RWIPosting{posting},
		},
		5*time.Minute,
	)

	if got := backoff.Longest(identity); got != 5*time.Minute {
		t.Fatalf("Longest = %v, want 5m", got)
	}
}

func TestPostingBackoffLeavesUnrelatedPostingUntouched(t *testing.T) {
	word1, url1 := yacymodel.WordHash("w1"), urlHash("u1")
	word2, url2 := yacymodel.WordHash("w2"), urlHash("u2")
	identity2 := postingschedule.Identity{Word: word2, URL: url2}
	peerOffer := offer{
		Peer: seed(yacymodel.WordHash("a")),
		Postings: []yacymodel.RWIPosting{
			fakePosting(word1, url1),
		},
	}
	backoff := newPostingBackoff()

	backoff.Record(peerOffer, time.Minute)

	if got := backoff.Longest(identity2); got != 0 {
		t.Fatalf("Longest = %v, want 0 for an unrelated posting", got)
	}
}
