package wordpartitionoffer_test

import (
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

func outcomeWordPartition() yacymodel.WordPartition {
	return yacymodel.WordPartition{Word: wordOf(0), Partition: fixturePartition()}
}

func postingsOf(urls ...yacymodel.URLHash) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(urls))
	for _, url := range urls {
		postings = append(
			postings,
			yacymodel.RWIPosting{WordHash: outcomeWordPartition().Word, URLHash: url},
		)
	}

	return postings
}

func undeliveredOf(urls ...yacymodel.URLHash) []undeliveredpostings.UndeliveredPosting {
	undelivered := make([]undeliveredpostings.UndeliveredPosting, 0, len(urls))
	for _, url := range urls {
		undelivered = append(
			undelivered,
			undeliveredpostings.UndeliveredPosting{WordPartition: outcomeWordPartition(), URL: url},
		)
	}

	return undelivered
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
	urls := urlsOfPartition(2)
	whole := postingsOf(urls...)
	for name, answer := range map[string]wordpartitionoffer.PeerAnswer{
		"accepted everything": accepted(whole),
		"accepted a part":     accepted(whole[:1]),
		"declined":            declined(),
	} {
		t.Run(name, func(t *testing.T) {
			missingHolder := outcomePeer(0)
			offer := wordpartitionoffer.WordPartitionOffer{
				WordPartition:    outcomeWordPartition(),
				ResponsiblePeers: []yacymodel.Hash{missingHolder.Hash},
				Recipients: []wordpartitionoffer.Recipient{
					{Peer: missingHolder, Postings: whole},
				},
			}

			outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
				missingHolder.Hash: answer,
			})

			becameHolder := slices.Contains(outcome.Holders, missingHolder.Hash)
			if wantHolder := len(
				answer.AcceptedPostings,
			) == len(
				whole,
			); becameHolder != wantHolder {
				t.Fatalf("Holders = %v, want holder %v", outcome.Holders, wantHolder)
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
	responsiblePeer, staleHolder := outcomePeer(0), outcomePeer(1<<40)
	whole := postingsOf(urlsOfPartition(1)...)
	offer := wordpartitionoffer.WordPartitionOffer{
		WordPartition:    outcomeWordPartition(),
		ResponsiblePeers: []yacymodel.Hash{responsiblePeer.Hash},
		Holders:          []yacymodel.Hash{staleHolder.Hash},
		StaleHolders:     []yacymodel.Hash{staleHolder.Hash},
		Recipients:       []wordpartitionoffer.Recipient{{Peer: responsiblePeer, Postings: whole}},
	}

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		responsiblePeer.Hash: declined(),
	})

	if len(outcome.Holders) != 0 || outcome.AmountOfDroppedHolders != 1 {
		t.Fatalf("Holders = %v, AmountOfDroppedHolders = %d, want no holders and one dropped",
			outcome.Holders, outcome.AmountOfDroppedHolders)
	}
}

func TestAPostingIsDeliveredOnceEveryResponsiblePeerAcceptedIt(t *testing.T) {
	urls := urlsOfPartition(2)
	whole := postingsOf(urls...)
	holder, missingHolder := outcomePeer(0), outcomePeer(1)
	offer := wordpartitionoffer.WordPartitionOffer{
		WordPartition:       outcomeWordPartition(),
		ResponsiblePeers:    []yacymodel.Hash{holder.Hash, missingHolder.Hash},
		Holders:             []yacymodel.Hash{holder.Hash},
		UndeliveredPostings: undeliveredOf(urls...),
		Recipients: []wordpartitionoffer.Recipient{
			{Peer: holder, Postings: whole},
			{Peer: missingHolder, Postings: whole},
		},
	}

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		holder.Hash:        accepted(whole),
		missingHolder.Hash: accepted(whole[:1]),
	})

	if got, want := outcome.DeliveredPostings, undeliveredOf(urls[0]); !slices.Equal(got, want) {
		t.Fatalf("DeliveredPostings = %v, want only %v", got, want)
	}
	if slices.Contains(outcome.Holders, missingHolder.Hash) ||
		!slices.Contains(outcome.Holders, holder.Hash) {
		t.Fatalf("Holders = %v, want only %v", outcome.Holders, holder.Hash)
	}
}

func TestAcceptancesCountThePostingsOfThisOfferEachRecipientAccepted(t *testing.T) {
	urls := urlsOfPartition(3)
	whole := postingsOf(urls...)
	otherWordPosting := yacymodel.RWIPosting{WordHash: wordOf(1), URLHash: urls[0]}
	accepting, declining := outcomePeer(0), outcomePeer(1)
	offer := wordpartitionoffer.WordPartitionOffer{
		WordPartition:    outcomeWordPartition(),
		ResponsiblePeers: []yacymodel.Hash{accepting.Hash, declining.Hash},
		Recipients: []wordpartitionoffer.Recipient{
			{Peer: accepting, Postings: whole},
			{Peer: declining, Postings: whole},
		},
	}

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		accepting.Hash: accepted(append(slices.Clone(whole[:2]), otherWordPosting)),
		declining.Hash: declined(),
	})

	want := []wordpartitionoffer.Acceptance{{Peer: accepting.Hash, AmountOfPostings: 2}}
	if !slices.Equal(outcome.Acceptances, want) {
		t.Fatalf("Acceptances = %v, want %v", outcome.Acceptances, want)
	}
}

func TestAWordPartitionIsAtRedundancyOnceEveryResponsiblePeerHoldsIt(t *testing.T) {
	whole := postingsOf(urlsOfPartition(1)...)
	holder, missingHolder := outcomePeer(0), outcomePeer(1)
	for name, scenario := range map[string]struct {
		missingHolderAnswer wordpartitionoffer.PeerAnswer
		wantAtRedundancy    bool
	}{
		"missing holder accepted": {missingHolderAnswer: accepted(whole), wantAtRedundancy: true},
		"missing holder declined": {missingHolderAnswer: declined()},
	} {
		t.Run(name, func(t *testing.T) {
			offer := wordpartitionoffer.WordPartitionOffer{
				WordPartition:    outcomeWordPartition(),
				ResponsiblePeers: []yacymodel.Hash{holder.Hash, missingHolder.Hash},
				Holders:          []yacymodel.Hash{holder.Hash},
				Recipients: []wordpartitionoffer.Recipient{
					{Peer: holder, Postings: whole},
					{Peer: missingHolder, Postings: whole},
				},
			}

			outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
				holder.Hash:        accepted(whole),
				missingHolder.Hash: scenario.missingHolderAnswer,
			})

			if outcome.IsAtRedundancy != scenario.wantAtRedundancy {
				t.Fatalf("IsAtRedundancy = %v with holders %v, want %v",
					outcome.IsAtRedundancy, outcome.Holders, scenario.wantAtRedundancy)
			}
		})
	}
}

func TestAWordPartitionIsShortOfRedundancyWhileAnOfferedPostingStaysUndelivered(t *testing.T) {
	urls := urlsOfPartition(1)
	newPostings := postingsOf(urls...)
	holder, decliningHolder := outcomePeer(0), outcomePeer(1)
	offer := wordpartitionoffer.WordPartitionOffer{
		WordPartition:       outcomeWordPartition(),
		ResponsiblePeers:    []yacymodel.Hash{holder.Hash, decliningHolder.Hash},
		Holders:             []yacymodel.Hash{holder.Hash, decliningHolder.Hash},
		UndeliveredPostings: undeliveredOf(urls...),
		Recipients: []wordpartitionoffer.Recipient{
			{Peer: holder, Postings: newPostings},
			{Peer: decliningHolder, Postings: newPostings},
		},
	}

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		holder.Hash:          accepted(newPostings),
		decliningHolder.Hash: declined(),
	})

	if !slices.Equal(outcome.Holders, offer.Holders) || outcome.IsAtRedundancy {
		t.Fatalf("Holders = %v, IsAtRedundancy = %v, want the same holders short of redundancy",
			outcome.Holders, outcome.IsAtRedundancy)
	}
}
