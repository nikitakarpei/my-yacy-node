package wordpartitionoffer_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

func outcomeWordPartition() yacymodel.WordPartition {
	return yacymodel.WordPartition{Word: wordOf(0), Partition: fixturePartition()}
}

func accepted(postings []yacymodel.RWIPosting) wordpartitionoffer.PeerAnswer {
	return wordpartitionoffer.PeerAnswer{Accepted: true, AcceptedPostings: postings}
}

func declined() wordpartitionoffer.PeerAnswer {
	return wordpartitionoffer.PeerAnswer{}
}

func outcomePeer(steps yacymodel.DHTRingPosition) yacymodel.Seed {
	return peerAfter(outcomeWordPartition(), steps)
}

func TestAMissingHolderBecomesAHolderOnlyIfItAcceptedEverythingSentToIt(t *testing.T) {
	for name, scenario := range map[string]struct {
		answerTo   func(sent []yacymodel.RWIPosting) wordpartitionoffer.PeerAnswer
		wantHolder bool
	}{
		"accepted everything": {answerTo: accepted, wantHolder: true},
		"accepted a part":     {answerTo: acceptedInPart},
		"declined":            {answerTo: declinedWhole},
	} {
		t.Run(name, func(t *testing.T) {
			h := openHarness(t, options{})
			missingHolder := outcomePeer(0)
			offer := firstOfferTo(t, h, missingHolder)

			outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
				missingHolder.Hash: scenario.answerTo(postingsSentTo(t, offer, missingHolder)),
			})

			if got := slices.Contains(
				outcome.Holders,
				missingHolder.Hash,
			); got != scenario.wantHolder {
				t.Fatalf("Holders = %v, want holder %v", outcome.Holders, scenario.wantHolder)
			}
		})
	}
}

type offerToPeer func(
	t *testing.T,
	h *harness,
	peer yacymodel.Seed,
) wordpartitionoffer.WordPartitionOffer

func firstOfferTo(
	t *testing.T,
	h *harness,
	missingHolder yacymodel.Seed,
) wordpartitionoffer.WordPartitionOffer {
	t.Helper()

	h.admit(t, outcomeWordPartition(), urlsOfPartition(2)...)

	return onlyOffer(t, h.undeliveredBatch(t, missingHolder).Offers)
}

func undeliveredOfferTo(
	t *testing.T,
	h *harness,
	holder yacymodel.Seed,
) wordpartitionoffer.WordPartitionOffer {
	t.Helper()

	h.admit(t, outcomeWordPartition(), urlsOfPartition(2)...)
	h.recordHolders(t, outcomeWordPartition(), holder.Hash)

	return onlyOffer(t, h.undeliveredBatch(t, holder).Offers)
}

func dueRefreshTo(
	t *testing.T,
	h *harness,
	holder yacymodel.Seed,
) wordpartitionoffer.WordPartitionOffer {
	t.Helper()

	h.admit(t, outcomeWordPartition(), urlsOfPartition(2)...)
	h.deliverEverything(t)
	h.recordHolders(t, outcomeWordPartition(), holder.Hash)
	h.clock = h.clock.Add(refreshInterval)

	return onlyOffer(t, h.refreshBatch(t, holder).Offers)
}

func acceptedInPart(sent []yacymodel.RWIPosting) wordpartitionoffer.PeerAnswer {
	return accepted(sent[:1])
}

func declinedWhole([]yacymodel.RWIPosting) wordpartitionoffer.PeerAnswer {
	return declined()
}

func TestARefreshDropsAHolderThatAnsweredWithoutAcceptingEverything(t *testing.T) {
	for name, scenario := range map[string]struct {
		offerTo    offerToPeer
		answerTo   func(sent []yacymodel.RWIPosting) wordpartitionoffer.PeerAnswer
		wantHolder bool
	}{
		"refresh accepted in part":     {offerTo: dueRefreshTo, answerTo: acceptedInPart},
		"refresh declined":             {offerTo: dueRefreshTo, answerTo: declinedWhole, wantHolder: true},
		"refresh accepted in full":     {offerTo: dueRefreshTo, answerTo: accepted, wantHolder: true},
		"refresh without an answer":    {offerTo: dueRefreshTo, wantHolder: true},
		"undelivered accepted in part": {offerTo: undeliveredOfferTo, answerTo: acceptedInPart, wantHolder: true},
		"undelivered declined":         {offerTo: undeliveredOfferTo, answerTo: declinedWhole, wantHolder: true},
	} {
		t.Run(name, func(t *testing.T) {
			h := openHarness(t, options{})
			holder := outcomePeer(0)
			offer := scenario.offerTo(t, h, holder)
			answers := map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{}
			if scenario.answerTo != nil {
				answers[holder.Hash] = scenario.answerTo(postingsSentTo(t, offer, holder))
			}

			outcome := offer.OutcomeFrom(answers)

			if got := slices.Contains(outcome.Holders, holder.Hash); got != scenario.wantHolder {
				t.Fatalf("Holders = %v, want holder %v", outcome.Holders, scenario.wantHolder)
			}
		})
	}
}

func TestAHolderThatGetsNoPostingsInThisBatchStaysAHolder(t *testing.T) {
	h := openHarness(t, options{})
	holder, unreachableHolder := outcomePeer(0), outcomePeer(1)
	h.admit(t, outcomeWordPartition(), urlsOfPartition(1)...)
	h.deliverEverything(t)
	h.recordHolders(t, outcomeWordPartition(), holder.Hash, unreachableHolder.Hash)
	h.reachability.recentlyReachable[unreachableHolder.Hash] = true
	h.clock = h.clock.Add(refreshInterval)
	refresh := onlyOffer(t, h.refreshBatch(t, holder).Offers)

	outcome := refresh.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		holder.Hash: accepted(postingsSentTo(t, refresh, holder)),
	})

	if want := []yacymodel.Hash{
		holder.Hash,
		unreachableHolder.Hash,
	}; !slices.Equal(
		outcome.Holders,
		want,
	) {
		t.Fatalf("Holders = %v, want %v unchanged", outcome.Holders, want)
	}
}

func TestHoldersAreRecordedUnlessAnUndeliveredOfferLeavesThemUnchangedAtRedundancy(t *testing.T) {
	for name, scenario := range map[string]struct {
		offerTo      offerToPeer
		answerTo     func(sent []yacymodel.RWIPosting) wordpartitionoffer.PeerAnswer
		wantRecorded bool
	}{
		"undelivered accepted by the holder":   {offerTo: undeliveredOfferTo, answerTo: accepted},
		"undelivered declined by the holder":   {offerTo: undeliveredOfferTo, answerTo: declinedWhole, wantRecorded: true},
		"undelivered accepted by a new holder": {offerTo: firstOfferTo, answerTo: accepted, wantRecorded: true},
		"refresh accepted by the holder":       {offerTo: dueRefreshTo, answerTo: accepted, wantRecorded: true},
	} {
		t.Run(name, func(t *testing.T) {
			h := openHarness(t, options{})
			peer := outcomePeer(0)
			offer := scenario.offerTo(t, h, peer)

			outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
				peer.Hash: scenario.answerTo(postingsSentTo(t, offer, peer)),
			})

			if outcome.MustRecordHolders != scenario.wantRecorded {
				t.Fatalf("MustRecordHolders = %v with holders %v, want %v",
					outcome.MustRecordHolders, outcome.Holders, scenario.wantRecorded)
			}
		})
	}
}

func TestStaleHoldersAreDroppedAndCounted(t *testing.T) {
	h := openHarness(t, options{})
	responsiblePeer, staleHolder := outcomePeer(0), outcomePeer(1<<40)
	h.admit(t, outcomeWordPartition(), urlsOfPartition(1)...)
	h.recordHolders(t, outcomeWordPartition(), staleHolder.Hash)
	offer := onlyOffer(t, h.undeliveredBatch(t, responsiblePeer).Offers)

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		responsiblePeer.Hash: declined(),
	})

	if len(outcome.Holders) != 0 || outcome.AmountOfStaleHolders != 1 {
		t.Fatalf("Holders = %v, AmountOfStaleHolders = %d, want no holders and one stale",
			outcome.Holders, outcome.AmountOfStaleHolders)
	}
}

func TestAPostingIsDeliveredOnceEveryResponsiblePeerAcceptedIt(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	holder, missingHolder := outcomePeer(0), outcomePeer(1)
	h.admit(t, outcomeWordPartition(), urlsOfPartition(2)...)
	h.recordHolders(t, outcomeWordPartition(), holder.Hash)
	offer := onlyOffer(t, h.undeliveredBatch(t, holder, missingHolder).Offers)
	sentToMissingHolder := postingsSentTo(t, offer, missingHolder)

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		holder.Hash:        accepted(postingsSentTo(t, offer, holder)),
		missingHolder.Hash: accepted(sentToMissingHolder[:1]),
	})

	want := []yacymodel.URLHash{sentToMissingHolder[0].URLHash}
	if got := urlsOfUndelivered(outcome.DeliveredPostings); !slices.Equal(got, want) {
		t.Fatalf("DeliveredPostings = %v, want only %v", got, want)
	}
	if want := []yacymodel.Hash{holder.Hash}; !slices.Equal(outcome.Holders, want) {
		t.Fatalf("Holders = %v, want only %v", outcome.Holders, want)
	}
}

func TestAcceptancesCountThePostingsOfThisOfferEachRecipientAccepted(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	acceptingPeer, decliningPeer := outcomePeer(0), outcomePeer(1)
	h.admit(t, outcomeWordPartition(), urlsOfPartition(3)...)
	offer := onlyOffer(t, h.undeliveredBatch(t, acceptingPeer, decliningPeer).Offers)
	sent := postingsSentTo(t, offer, acceptingPeer)
	otherWordPosting := yacymodel.RWIPosting{WordHash: wordOf(1), URLHash: sent[2].URLHash}

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		acceptingPeer.Hash: accepted(append(slices.Clone(sent[:2]), otherWordPosting)),
		decliningPeer.Hash: declined(),
	})

	want := []wordpartitionoffer.Acceptance{{Peer: acceptingPeer.Hash, AmountOfPostings: 2}}
	if !slices.Equal(outcome.Acceptances, want) {
		t.Fatalf("Acceptances = %v, want %v", outcome.Acceptances, want)
	}
}

func TestAWordPartitionIsAtRedundancyOnceEveryResponsiblePeerHoldsIt(t *testing.T) {
	for name, scenario := range map[string]struct {
		missingHolderAnswerTo func(sent []yacymodel.RWIPosting) wordpartitionoffer.PeerAnswer
		wantAtRedundancy      bool
	}{
		"missing holder accepted": {missingHolderAnswerTo: accepted, wantAtRedundancy: true},
		"missing holder declined": {missingHolderAnswerTo: declinedWhole},
	} {
		t.Run(name, func(t *testing.T) {
			h := openHarness(t, options{redundancy: 2})
			holder, missingHolder := outcomePeer(0), outcomePeer(1)
			h.admit(t, outcomeWordPartition(), urlsOfPartition(1)...)
			h.recordHolders(t, outcomeWordPartition(), holder.Hash)
			offer := onlyOffer(t, h.undeliveredBatch(t, holder, missingHolder).Offers)

			outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
				holder.Hash: accepted(postingsSentTo(t, offer, holder)),
				missingHolder.Hash: scenario.missingHolderAnswerTo(
					postingsSentTo(t, offer, missingHolder),
				),
			})

			if outcome.IsAtRedundancy != scenario.wantAtRedundancy {
				t.Fatalf("IsAtRedundancy = %v with holders %v, want %v",
					outcome.IsAtRedundancy, outcome.Holders, scenario.wantAtRedundancy)
			}
		})
	}
}

func TestAWordPartitionIsShortOfRedundancyWhileAnOfferedPostingStaysUndelivered(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	holder, decliningHolder := outcomePeer(0), outcomePeer(1)
	h.admit(t, outcomeWordPartition(), urlsOfPartition(1)...)
	h.recordHolders(t, outcomeWordPartition(), holder.Hash, decliningHolder.Hash)
	offer := onlyOffer(t, h.undeliveredBatch(t, holder, decliningHolder).Offers)

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		holder.Hash:          accepted(postingsSentTo(t, offer, holder)),
		decliningHolder.Hash: declined(),
	})

	want := []yacymodel.Hash{holder.Hash, decliningHolder.Hash}
	if !slices.Equal(outcome.Holders, want) || outcome.IsAtRedundancy {
		t.Fatalf("Holders = %v, IsAtRedundancy = %v, want %v short of redundancy",
			outcome.Holders, outcome.IsAtRedundancy, want)
	}
}
