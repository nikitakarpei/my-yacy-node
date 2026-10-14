package distributioncycle_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestCycleTakesTheSectorsInRingOrder(t *testing.T) {
	later, earlier := wordPartitionsIn(40, 1)[0], wordPartitionsIn(3, 1)[0]
	h := openSectorCycle(t, startClock(), cycleOptions{})
	h.admit(t, later, urlsOf(1)...)
	h.admit(t, earlier, urlsOf(1)...)

	h.runCycle(t)

	if len(h.transfers.offered) != 2 ||
		h.transfers.offered[0].Postings[0].WordHash != earlier.Word ||
		h.transfers.offered[1].Postings[0].WordHash != later.Word {
		t.Fatalf("offered %+v, want sector 3 before sector 40", h.transfers.offered)
	}
}

func TestCycleOffersASectorAgainWhileItHoldsMoreThanABatch(t *testing.T) {
	wordPartitions := wordPartitionsIn(3, postingsPerBatch+1)
	h := openSectorCycle(t, startClock(), cycleOptions{})
	for _, wordPartition := range wordPartitions {
		h.admit(t, wordPartition, urlsOf(1)...)
	}

	h.runCycle(t)

	if len(h.transfers.offered) != 2 || len(h.transfers.offered[1].Postings) != 1 {
		t.Fatalf("offered %+v, want a full batch, then the one posting left", h.transfers.offered)
	}
	assertEachWordPartitionOfferedOnce(t, h.transfers.offered, wordPartitions)
}

func TestCycleContinuesAfterTheLastSectorItTook(t *testing.T) {
	firstSector := wordPartitionsIn(3, postingsPerBatch+1)
	nextSector := wordPartitionsIn(5, 1)[0]
	clk := startClock()
	h := openSectorCycle(t, clk, cycleOptions{drainBudget: 30 * time.Second})
	h.transfers.onOffer = func() { clk.at = clk.at.Add(30 * time.Second) }
	for _, wordPartition := range append(firstSector, nextSector) {
		h.admit(t, wordPartition, urlsOf(1)...)
	}

	h.runCycle(t)
	h.runCycle(t)

	if len(h.transfers.offered) != 2 ||
		h.transfers.offered[1].Postings[0].WordHash != nextSector.Word {
		t.Fatalf(
			"offered %+v, want the second cycle to start at the next sector",
			h.transfers.offered,
		)
	}
}

func openSectorCycle(t *testing.T, clk *clock, opts cycleOptions) *cycleHarness {
	t.Helper()

	opts.redundancy = 2
	h := openCycle(t, clk, opts)
	peer := seed(yacymodel.WordHash("peer"))
	h.reach(peer)
	h.transfers.answers[peer.Hash] = acceptedAnswer

	return h
}
