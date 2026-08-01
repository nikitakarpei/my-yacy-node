package rwidistribution

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestAddGroupsPostingsByPeer(t *testing.T) {
	byPeer := newPostingOffersByPeer()
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

func TestFullReportsPeerAtCap(t *testing.T) {
	byPeer := newPostingOffersByPeer()
	peer := seed(yacymodel.WordHash("peer"))
	word := yacymodel.WordHash("w1")
	posting := fakePosting(word, urlHash("u1"))

	for i := range postingsPerPeerCap {
		if byPeer.Full(peer.Hash) {
			t.Fatalf("Full = true at %v, want false below the cap", i)
		}
		byPeer.Add(peer, posting)
	}

	if !byPeer.Full(peer.Hash) {
		t.Fatalf("Full = false, want true once the peer holds %v postings", postingsPerPeerCap)
	}

	offers := byPeer.Offers()
	if len(offers) != 1 || len(offers[0].Postings) != postingsPerPeerCap {
		t.Fatalf("offers = %v postings, want %v", len(offers[0].Postings), postingsPerPeerCap)
	}
}
