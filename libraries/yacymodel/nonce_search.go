package yacymodel

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type nonceSearch struct {
	keyDigest [sha256.Size]byte
	stopped   atomic.Bool

	mu        sync.Mutex
	timeOver  bool
	bestNonce uint64
	bestWork  int
}

func bestNonceFor(
	ctx context.Context,
	keyDigest [sha256.Size]byte,
	searchTime time.Duration,
	clock Clock,
) (uint64, error) {
	search := &nonceSearch{keyDigest: keyDigest, bestWork: identityWorkMinimum - 1}
	stopOnCancel := context.AfterFunc(ctx, search.stop)
	defer stopOnCancel()
	stopSearchTime := clock.After(searchTime, search.endTime)
	defer stopSearchTime()
	search.runEach(runtime.GOMAXPROCS(0))
	if err := context.Cause(ctx); err != nil {
		return 0, fmt.Errorf("search the peer key nonce: %w", err)
	}

	return search.bestNonce, nil
}

func (s *nonceSearch) stop() {
	s.stopped.Store(true)
}

func (s *nonceSearch) endTime() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.timeOver = true
	if s.bestWork >= identityWorkMinimum {
		s.stop()
	}
}

func (s *nonceSearch) runEach(searchCount int) {
	var searches sync.WaitGroup
	searchSpan := math.MaxUint64 / uint64(max(searchCount, 1))
	var firstNonce uint64
	for range searchCount {
		searchStart := firstNonce
		searches.Go(func() { s.searchFrom(searchStart) })
		firstNonce += searchSpan
	}
	searches.Wait()
}

func (s *nonceSearch) searchFrom(firstNonce uint64) {
	bestWork := identityWorkMinimum - 1
	for nonce := firstNonce; !s.stopped.Load(); nonce++ {
		if work := workOf(puzzleDigestFrom(s.keyDigest, nonce)); work > bestWork {
			bestWork = work
			s.record(nonce, work)
		}
	}
}

func (s *nonceSearch) record(nonce uint64, work int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if work > s.bestWork {
		s.bestNonce, s.bestWork = nonce, work
	}
	if s.timeOver {
		s.stop()
	}
}
