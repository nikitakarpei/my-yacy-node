package rwidistribution

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestAddGroupsPostingsByPeer(t *testing.T) {
	batch := newPostingOfferBatch()
	peer := seed(yacymodel.WordHash("peer"))
	word := yacymodel.WordHash("w1")

	for _, url := range []string{"u1", "u2"} {
		if !batch.Add(peer, fakePosting(word, urlHash(url))) {
			t.Fatalf("Add %v = false, want true", url)
		}
	}

	offers := batch.Offers()
	if len(offers) != 1 || offers[0].Peer.Hash != peer.Hash || len(offers[0].Postings) != 2 {
		t.Fatalf("offers = %+v, want one offer to %v with 2 postings", offers, peer.Hash)
	}
}

func TestAddRefusesPostingBeyondPeerCap(t *testing.T) {
	batch := newPostingOfferBatch()
	peer := seed(yacymodel.WordHash("peer"))
	word := yacymodel.WordHash("w1")
	posting := fakePosting(word, urlHash("u1"))

	for i := range postingsPerPeerCap {
		if !batch.Add(peer, posting) {
			t.Fatalf("Add %v = false, want true below the cap", i)
		}
	}

	if batch.Add(peer, posting) {
		t.Fatalf("Add = true, want false once the peer holds %v postings", postingsPerPeerCap)
	}

	offers := batch.Offers()
	if len(offers) != 1 || len(offers[0].Postings) != postingsPerPeerCap {
		t.Fatalf("offers = %v postings, want %v", len(offers[0].Postings), postingsPerPeerCap)
	}
}
