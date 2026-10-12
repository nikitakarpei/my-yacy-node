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
			); becameHolder != wantHolder ||
				outcome.HoldersChanged != wantHolder {
				t.Fatalf("Holders = %v, HoldersChanged = %v, want holder %v",
					outcome.Holders, outcome.HoldersChanged, wantHolder)
			}
		})
	}
}

func TestARefreshDropsAHolderThatAnsweredWithoutAcceptingEverything(t *testing.T) {
	whole := postingsOf(urlsOfPartition(2)...)
	holder := outcomePeer(0)
	answersOf := func(answer wordpartitionoffer.PeerAnswer) map[yacymodel.Hash]wordpartitionoffer.PeerAnswer {
		return map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{holder.Hash: answer}
	}
	for name, scenario := range map[string]struct {
		isRefresh  bool
		answers    map[yacymodel.Hash]wordpartitionoffer.PeerAnswer
		wantHolder bool
	}{
		"refresh accepted in part":     {isRefresh: true, answers: answersOf(accepted(whole[:1]))},
		"refresh declined":             {isRefresh: true, answers: answersOf(declined()), wantHolder: true},
		"refresh accepted in full":     {isRefresh: true, answers: answersOf(accepted(whole)), wantHolder: true},
		"refresh without an answer":    {isRefresh: true, wantHolder: true},
		"undelivered accepted in part": {answers: answersOf(accepted(whole[:1])), wantHolder: true},
		"undelivered declined":         {answers: answersOf(declined()), wantHolder: true},
	} {
		t.Run(name, func(t *testing.T) {
			offer := wordpartitionoffer.WordPartitionOffer{
				WordPartition:    outcomeWordPartition(),
				ResponsiblePeers: []yacymodel.Hash{holder.Hash},
				Holders:          []yacymodel.Hash{holder.Hash},
				IsRefresh:        scenario.isRefresh,
				Recipients:       []wordpartitionoffer.Recipient{{Peer: holder, Postings: whole}},
			}

			outcome := offer.OutcomeFrom(scenario.answers)

			if got := slices.Contains(outcome.Holders, holder.Hash); got != scenario.wantHolder ||
				outcome.HoldersChanged == scenario.wantHolder {
				t.Fatalf("Holders = %v, HoldersChanged = %v, want holder %v",
					outcome.Holders, outcome.HoldersChanged, scenario.wantHolder)
			}
		})
	}
}

func TestAHolderThatGetsNoPostingsInThisBatchStaysAHolder(t *testing.T) {
	holder, unreachableHolder := outcomePeer(0), outcomePeer(1)
	offer := wordpartitionoffer.WordPartitionOffer{
		WordPartition:    outcomeWordPartition(),
		ResponsiblePeers: []yacymodel.Hash{holder.Hash},
		Holders:          []yacymodel.Hash{holder.Hash, unreachableHolder.Hash},
		IsRefresh:        true,
		Recipients: []wordpartitionoffer.Recipient{
			{Peer: holder, Postings: postingsOf(urlsOfPartition(1)...)},
		},
	}

	outcome := offer.OutcomeFrom(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer{
		holder.Hash: accepted(postingsOf(urlsOfPartition(1)...)),
	})

	if !slices.Equal(outcome.Holders, offer.Holders) || outcome.HoldersChanged {
		t.Fatalf("Holders = %v, HoldersChanged = %v, want %v unchanged",
			outcome.Holders, outcome.HoldersChanged, offer.Holders)
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

	if len(outcome.Holders) != 0 || !outcome.HoldersChanged || outcome.AmountOfDroppedHolders != 1 {
		t.Fatalf("Holders = %v, HoldersChanged = %v, AmountOfDroppedHolders = %d, "+
			"want no holders and one dropped",
			outcome.Holders, outcome.HoldersChanged, outcome.AmountOfDroppedHolders)
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

	if outcome.HoldersChanged || outcome.IsAtRedundancy {
		t.Fatalf("HoldersChanged = %v, IsAtRedundancy = %v, want the same holders short of "+
			"redundancy", outcome.HoldersChanged, outcome.IsAtRedundancy)
	}
}
