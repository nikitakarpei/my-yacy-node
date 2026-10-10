package distributioncycle_test

import (
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
)

func TestCycleTakesTheSectorsInRingOrder(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	earlierDue, laterDue := postingsInSector(40, 1), postingsInSector(3, 1)
	clk := &clock{at: time.Unix(1000, 0)}
	h := openCycle(t, clk, cycleOptions{
		postings:   postingsByKey(slices.Concat(earlierDue, laterDue)),
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy: 2,
	})
	h.courier.receipts[peer] = postingcourier.Receipt{Outcome: postingcourier.Accepted}
	store(t, h.v, h.schedule, earlierDue[0].WordHash, earlierDue[0].URLHash)
	clk.at = clk.at.Add(time.Minute)
	store(t, h.v, h.schedule, laterDue[0].WordHash, laterDue[0].URLHash)

	h.runCycle(t)

	if len(h.courier.offered) != 2 ||
		!samePostings(h.courier.offered[0].Postings, laterDue) ||
		!samePostings(h.courier.offered[1].Postings, earlierDue) {
		t.Fatalf("offered %+v, want sector 3 before sector 40", h.courier.offered)
	}
}

func TestCycleOffersASectorAgainWhileItHoldsMoreThanABatch(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	sectorPostings := postingsInSector(3, postingsPerBatch+1)
	h := openCycle(t, &clock{at: time.Unix(1000, 0)}, cycleOptions{
		postings:   postingsByKey(sectorPostings),
		roster:     fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy: 2,
	})
	h.courier.receipts[peer] = postingcourier.Receipt{Outcome: postingcourier.Accepted}
	storePostings(t, h, sectorPostings)

	h.runCycle(t)

	if len(h.courier.offered) != 2 || len(h.courier.offered[1].Postings) != 1 {
		t.Fatalf("offered %+v, want a full batch, then the one posting left", h.courier.offered)
	}
	assertEachPostingOfferedOnce(t, h.courier.offered, sectorPostings)
}

func TestCycleContinuesAfterTheLastSectorItTook(t *testing.T) {
	peer := yacymodel.WordHash("peer")
	firstSector := postingsInSector(3, postingsPerBatch+1)
	nextSector := postingsInSector(5, 1)
	clk := &clock{at: time.Unix(1000, 0)}
	h := openCycle(t, clk, cycleOptions{
		postings:    postingsByKey(slices.Concat(firstSector, nextSector)),
		roster:      fakeRoster{reachable: []yacymodel.Seed{seed(peer)}},
		redundancy:  2,
		drainBudget: 30 * time.Second,
	})
	h.courier.receipts[peer] = postingcourier.Receipt{Outcome: postingcourier.Accepted}
	h.courier.onOffer = func() { clk.at = clk.at.Add(30 * time.Second) }
	storePostings(t, h, slices.Concat(firstSector, nextSector))

	h.runCycle(t)
	h.runCycle(t)

	if len(h.courier.offered) != 2 || !samePostings(h.courier.offered[1].Postings, nextSector) {
		t.Fatalf(
			"offered %+v, want the second cycle to start at the next sector",
			h.courier.offered,
		)
	}
}

func storePostings(t *testing.T, h *cycleHarness, postings []yacymodel.RWIPosting) {
	t.Helper()

	for _, posting := range postings {
		store(t, h.v, h.schedule, posting.WordHash, posting.URLHash)
	}
}

func samePostings(postings, otherPostings []yacymodel.RWIPosting) bool {
	return len(postings) == len(otherPostings) &&
		!slices.ContainsFunc(postings, func(posting yacymodel.RWIPosting) bool {
			return !slices.Contains(otherPostings, posting)
		})
}
