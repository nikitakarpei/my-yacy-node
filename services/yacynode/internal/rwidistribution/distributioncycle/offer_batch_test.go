package distributioncycle

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestAddGroupsPostingsByPeer(t *testing.T) {
	byPeer := newOfferBatch()
	peer := seed(yacymodel.WordHash("peer"))
	word := yacymodel.WordHash("w1")

	for _, url := range []string{"u1", "u2"} {
		byPeer.Add(peer, fakePosting(word, urlHash(url)))
	}

	offers := byPeer.Offers()
	if len(offers) != 1 || offers[0].Peer.Hash != peer.Hash || len(offers[0].Postings) != 2 {
		t.Fatalf("offers = %+v, want one offer to %v with 2 postings", offers, peer.Hash)
	}
}

func TestAddRefusesPeerAtCap(t *testing.T) {
	byPeer := newOfferBatch()
	peer := seed(yacymodel.WordHash("peer"))
	word := yacymodel.WordHash("w1")
	posting := fakePosting(word, urlHash("u1"))

	for i := range postingsPerPeerCap {
		if !byPeer.Add(peer, posting) {
			t.Fatalf("Add = false at %v, want true below the cap", i)
		}
	}

	if byPeer.Add(peer, posting) {
		t.Fatalf("Add = true, want false once the peer holds %v postings", postingsPerPeerCap)
	}

	offers := byPeer.Offers()
	if len(offers) != 1 || len(offers[0].Postings) != postingsPerPeerCap {
		t.Fatalf("offers = %v postings, want %v", len(offers[0].Postings), postingsPerPeerCap)
	}
}
