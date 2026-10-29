package peerroster

import (
	"slices"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type reachablePeer struct {
	seed        yacymodel.Seed
	confirmedAt time.Time
}

type reachablePeers struct {
	capacity int

	mu    sync.Mutex
	peers map[yacymodel.Hash]reachablePeer
}

func newReachablePeers(capacity int) *reachablePeers {
	return &reachablePeers{capacity: capacity, peers: make(map[yacymodel.Hash]reachablePeer)}
}

func (s *reachablePeers) admit(
	seed yacymodel.Seed,
	confirmedAt time.Time,
) (admitted, wasReachable bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, wasReachable = s.peers[seed.Hash]
	admitted = wasReachable || len(s.peers) < s.capacity
	if admitted {
		s.peers[seed.Hash] = reachablePeer{seed: seed, confirmedAt: confirmedAt}
	}

	return admitted, wasReachable
}

func (s *reachablePeers) evict(peer yacymodel.Hash) (wasReachable bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, wasReachable = s.peers[peer]
	if wasReachable {
		delete(s.peers, peer)
	}

	return wasReachable
}

func (s *reachablePeers) contains(peer yacymodel.Hash) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, reachable := s.peers[peer]

	return reachable
}

func (s *reachablePeers) amount() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.peers)
}

func (s *reachablePeers) hashSet() map[yacymodel.Hash]struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	hashes := make(map[yacymodel.Hash]struct{}, len(s.peers))
	for hash := range s.peers {
		hashes[hash] = struct{}{}
	}

	return hashes
}

func (s *reachablePeers) seeds() []yacymodel.Seed {
	s.mu.Lock()
	defer s.mu.Unlock()

	seeds := make([]yacymodel.Seed, 0, len(s.peers))
	for _, reachable := range s.peers {
		seeds = append(seeds, reachable.seed)
	}

	return seeds
}

func (s *reachablePeers) mostRecentSeeds(limit int) []yacymodel.Seed {
	if limit <= 0 {
		return nil
	}
	ranked := s.byRecency()
	seeds := make([]yacymodel.Seed, 0, min(limit, len(ranked)))
	for _, reachable := range ranked[:min(limit, len(ranked))] {
		seeds = append(seeds, reachable.seed)
	}

	return seeds
}

func (s *reachablePeers) byRecency() []reachablePeer {
	s.mu.Lock()
	defer s.mu.Unlock()

	ranked := make([]reachablePeer, 0, len(s.peers))
	for _, reachable := range s.peers {
		ranked = append(ranked, reachable)
	}
	slices.SortFunc(ranked, func(a, b reachablePeer) int {
		return b.confirmedAt.Compare(a.confirmedAt)
	})

	return ranked
}
