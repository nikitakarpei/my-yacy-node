package distributionloop_test

import (
	"testing"
	"testing/synctest"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestLoopTakesTheSectorsInRingOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		later, earlier := wordPartitionsIn(40, 1)[0], wordPartitionsIn(3, 1)[0]
		h := openSectorLoop(t)
		h.admit(t, later, urlsOf(1)...)
		h.admit(t, earlier, urlsOf(1)...)

		h.start(t)

		if len(h.transfers.offered) != 2 ||
			h.transfers.offered[0].Postings[0].WordHash != earlier.Word ||
			h.transfers.offered[1].Postings[0].WordHash != later.Word {
			t.Fatalf("offered %+v, want sector 3 before sector 40", h.transfers.offered)
		}
	})
}

func TestLoopOffersASectorAgainWhileItHoldsMoreThanABatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, postingsPerBatch+1)
		h := openSectorLoop(t)
		for _, wordPartition := range wordPartitions {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if len(h.transfers.offered) != 2 || len(h.transfers.offered[1].Postings) != 1 {
			t.Fatalf(
				"offered %+v, want a full batch, then the one posting left",
				h.transfers.offered,
			)
		}
		assertEachWordPartitionOfferedOnce(t, h.transfers.offered, wordPartitions)
	})
}

func TestLoopTakesTheNextSectorBeforeItContinuesASector(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		firstSector := wordPartitionsIn(3, postingsPerBatch+1)
		nextSector := wordPartitionsIn(5, 1)[0]
		h := openSectorLoop(t)
		for _, wordPartition := range append(firstSector, nextSector) {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if len(h.transfers.offered) != 3 ||
			h.transfers.offered[1].Postings[0].WordHash != nextSector.Word ||
			h.transfers.offered[2].Postings[0].WordHash != firstSector[postingsPerBatch].Word {
			t.Fatalf(
				"offered %+v, want the next sector before the rest of the first one",
				h.transfers.offered,
			)
		}
	})
}

func TestLoopReadsASectorAgainOnlyAfterTheRescanDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wordPartitions := wordPartitionsIn(3, 2)
		h := openSectorLoop(t)
		h.admit(t, wordPartitions[0], urlsOf(1)...)
		h.start(t)

		h.admit(t, wordPartitions[1], urlsOf(1)...)
		h.advance(sectorRescanDelay - time.Millisecond)
		if len(h.transfers.offered) != 1 {
			t.Fatalf(
				"offered %+v, want the sector unread before the rescan delay",
				h.transfers.offered,
			)
		}
		h.advance(time.Millisecond)

		if len(h.transfers.offered) != 2 ||
			h.transfers.offered[1].Postings[0].WordHash != wordPartitions[1].Word {
			t.Fatalf(
				"offered %+v, want the new posting after the rescan delay",
				h.transfers.offered,
			)
		}
	})
}

func TestLoopReadsASectorAgainAtOnceAfterABatchThatEndedNoPass(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const postingsPerSmallBatch = 2
		wordPartitions := wordPartitionsIn(3, 10*postingsPerSmallBatch+1)
		lastWordPartition := wordPartitions[len(wordPartitions)-1]
		peerInCooldown := peerAfter(wordPartitions[len(wordPartitions)-2], 1)
		peer := peerAfter(lastWordPartition, 1)
		h := openLoop(t, startClock(), loopOptions{
			self:             thisNodeFartherThan(wordPartitions[0]),
			cooldown:         time.Hour,
			postingsPerBatch: postingsPerSmallBatch,
		})
		h.reach(peerInCooldown, peer)
		h.transfers.answers[peer.Hash] = acceptedAnswer
		h.eligibility.OfferDeclined(peerInCooldown.Hash, 0)
		for _, wordPartition := range wordPartitions {
			h.admit(t, wordPartition, urlsOf(1)...)
		}

		h.start(t)

		if got := h.holdersOf(t, lastWordPartition); len(got) != 1 || got[0] != peer.Hash {
			t.Fatalf("holders = %v, want %v without waiting for the rescan delay", got, peer.Hash)
		}
	})
}

func openSectorLoop(t *testing.T) *loopHarness {
	t.Helper()

	h := openLoop(t, startClock(), loopOptions{redundancy: 2})
	peer := seed(yacymodel.WordHash("peer"))
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer

	return h
}
