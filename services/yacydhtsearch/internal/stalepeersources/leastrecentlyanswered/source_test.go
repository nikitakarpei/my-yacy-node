package leastrecentlyanswered_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalepeersources/leastrecentlyanswered"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func hashOf(t *testing.T, symbol byte) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(string([]byte{
		symbol, symbol, symbol, symbol, symbol, symbol,
		symbol, symbol, symbol, symbol, symbol, symbol,
	}))
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return hash
}

func TestThePeerThatAnsweredLongestAgoIsTheStalest(t *testing.T) {
	t.Parallel()

	recent, old := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := leastrecentlyanswered.New().StalestPeers([]peerdirectory.KnownPeer{
		{Hash: recent, AnsweredAt: time.Unix(200, 0)},
		{Hash: old, AnsweredAt: time.Unix(100, 0)},
	}, 1)

	if len(stalest) != 1 || stalest[0] != old {
		t.Fatalf("StalestPeers = %v, want %s", stalest, old)
	}
}

func TestPeersThatNeverAnsweredAreRankedByAdmission(t *testing.T) {
	t.Parallel()

	early, late := hashOf(t, 'a'), hashOf(t, 'b')
	stalest := leastrecentlyanswered.New().StalestPeers([]peerdirectory.KnownPeer{
		{Hash: late, AdmittedAt: time.Unix(200, 0)},
		{Hash: early, AdmittedAt: time.Unix(100, 0)},
	}, 2)

	if len(stalest) != 2 || stalest[0] != early || stalest[1] != late {
		t.Fatalf("StalestPeers = %v, want %s then %s", stalest, early, late)
	}
}

func TestNoPeerIsStaleWhenNoneWasAskedFor(t *testing.T) {
	t.Parallel()

	known := []peerdirectory.KnownPeer{{Hash: hashOf(t, 'a')}}

	if stalest := leastrecentlyanswered.New().StalestPeers(known, 0); stalest != nil {
		t.Fatalf("StalestPeers = %v, want none", stalest)
	}
}
