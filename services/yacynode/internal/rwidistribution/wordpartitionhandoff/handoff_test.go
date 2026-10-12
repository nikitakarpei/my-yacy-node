package wordpartitionhandoff_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionhandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const (
	partitionExponent  = 2
	refreshInterval    = time.Hour
	positionStep       = 8
	farAwaySteps       = 1000
	handoffRedundancy  = 1
	postingsPerFixture = 3
)

type harness struct {
	vault       *vault.Vault
	index       rwipostings.PostingIndex
	admitter    rwipostings.PostingAdmitter
	undelivered *undeliveredpostings.UndeliveredPostings
	holders     *wordpartitionholders.Holders
	handoff     *wordpartitionhandoff.Handoff
}

func openHarness(t *testing.T, self yacymodel.Hash, redundancy int) *harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	undelivered, projection, err := undeliveredpostings.Open(v, partitions())
	if err != nil {
		t.Fatalf("undeliveredpostings.Open: %v", err)
	}
	index, admitter, purger, err := rwipostings.Open(
		context.Background(),
		v,
		partitions(),
		[]vault.Projection[yacymodel.RWIPosting]{projection},
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	holders, err := wordpartitionholders.Open(v, partitions(), refreshInterval, fixedNow)
	if err != nil {
		t.Fatalf("wordpartitionholders.Open: %v", err)
	}

	return &harness{
		vault:       v,
		index:       index,
		admitter:    admitter,
		undelivered: undelivered,
		holders:     holders,
		handoff: wordpartitionhandoff.New(index, purger, undelivered, holders,
			wordpartitionhandoff.Config{
				Partitions: partitions(),
				Self:       self,
				Redundancy: redundancy,
			}),
	}
}

func fixedNow() time.Time {
	return time.Unix(1_700_000_000, 0).UTC()
}

func (h *harness) admit(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	urls ...yacymodel.URLHash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, url := range urls {
			if err := h.admitter.Admit(tx, yacymodel.RWIPosting{
				WordHash: wordPartition.Word,
				URLHash:  url,
				Language: yacymodel.LanguageOfUndeclaredDocument,
			}); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *harness) deliver(t *testing.T, wordPartitions ...yacymodel.WordPartition) {
	t.Helper()

	var undelivered []undeliveredpostings.UndeliveredPosting
	h.read(t, func(tx *vault.Txn) error {
		for _, wordPartition := range wordPartitions {
			if err := h.undelivered.PostingsOf(
				tx,
				wordPartition,
				func(posting undeliveredpostings.UndeliveredPosting) (bool, error) {
					undelivered = append(undelivered, posting)

					return true, nil
				},
			); err != nil {
				return err
			}
		}

		return nil
	})
	h.write(t, func(tx *vault.Txn) error {
		for _, posting := range undelivered {
			if _, err := h.undelivered.MarkDelivered(tx, posting); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *harness) recordHolders(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	holders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.Record(tx, wordPartition, holders)
	})
}

func (h *harness) handOff(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	holders []yacymodel.Hash,
	responsiblePeers []yacymodel.Hash,
) bool {
	t.Helper()

	var wasHandedOff bool
	h.write(t, func(tx *vault.Txn) error {
		handedOff, err := h.handoff.HandOffIfHeldByResponsiblePeers(
			context.Background(), tx, wordPartition, holders, responsiblePeers,
		)
		wasHandedOff = handedOff

		return err
	})

	return wasHandedOff
}

func (h *harness) postingsOf(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
) []yacymodel.URLHash {
	t.Helper()

	var urls []yacymodel.URLHash
	h.read(t, func(tx *vault.Txn) error {
		return h.index.PostingsOfWordIn(
			tx,
			wordPartition.Word,
			wordPartition.Partition,
			func(posting yacymodel.RWIPosting) (bool, error) {
				urls = append(urls, posting.URLHash)

				return true, nil
			},
		)
	})

	return urls
}

func (h *harness) holdersOf(t *testing.T, wordPartition yacymodel.WordPartition) []yacymodel.Hash {
	t.Helper()

	var recordedHolders []yacymodel.Hash
	h.read(t, func(tx *vault.Txn) error {
		holders, err := h.holders.Of(tx, wordPartition)
		recordedHolders = holders

		return err
	})

	return recordedHolders
}

func (h *harness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h *harness) read(t *testing.T, view func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.View(context.Background(), view); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func partitions() yacymodel.DHTRingPartitions {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		panic(err)
	}

	return partitions
}

func fixtureWordPartition() yacymodel.WordPartition {
	return yacymodel.WordPartition{Word: wordOf(0), Partition: 0}
}

func sameWordOtherPartition() yacymodel.WordPartition {
	return yacymodel.WordPartition{Word: wordOf(0), Partition: 1}
}

func otherWordSamePartition() yacymodel.WordPartition {
	return yacymodel.WordPartition{Word: wordOf(1), Partition: 0}
}

func wordOf(word int) yacymodel.Hash {
	return yacymodel.WordHash("word" + strconv.Itoa(word))
}

func urlsIn(partition uint, amount int) []yacymodel.URLHash {
	var urls []yacymodel.URLHash
	for document := 0; len(urls) < amount; document++ {
		if url := urlHashOf(document); partitions().PartitionOf(url) == partition {
			urls = append(urls, url)
		}
	}

	return urls
}

func urlHashOf(document int) yacymodel.URLHash {
	url, err := yacymodel.ParseURLHash(yacymodel.WordHash("url" + strconv.Itoa(document)).String())
	if err != nil {
		panic(err)
	}

	return url
}

func peerAfter(
	wordPartition yacymodel.WordPartition,
	steps yacymodel.DHTRingPosition,
) yacymodel.Hash {
	return yacymodel.HashFromDHTRingPosition(
		yacymodel.DHTRingPositionOfWordPartition(wordPartition, partitions()) + steps*positionStep,
	)
}

func closerPeer() yacymodel.Hash { return peerAfter(fixtureWordPartition(), 2) }

func otherCloserPeer() yacymodel.Hash { return peerAfter(fixtureWordPartition(), 3) }

func thisNodeFartherThanEveryPeer() yacymodel.Hash {
	return peerAfter(fixtureWordPartition(), farAwaySteps)
}

func thisNodeCloserThanEveryPeer() yacymodel.Hash { return peerAfter(fixtureWordPartition(), 1) }

func deliveredFixture(t *testing.T, self yacymodel.Hash, redundancy int) *harness {
	t.Helper()

	h := openHarness(t, self, redundancy)
	h.admit(
		t,
		fixtureWordPartition(),
		urlsIn(fixtureWordPartition().Partition, postingsPerFixture)...)
	h.deliver(t, fixtureWordPartition())
	h.recordHolders(t, fixtureWordPartition(), closerPeer())

	return h
}

func assertKept(t *testing.T, h *harness, wordPartition yacymodel.WordPartition, amount int) {
	t.Helper()

	if held := h.postingsOf(t, wordPartition); len(held) != amount {
		t.Fatalf("postings of %+v = %v, want %d kept", wordPartition, held, amount)
	}
	if holders := h.holdersOf(t, wordPartition); len(holders) == 0 {
		t.Fatalf("holders of %+v forgotten, want them kept", wordPartition)
	}
}

func TestWordPartitionHandedOffOnceResponsiblePeersCloserThanThisNodeHoldIt(t *testing.T) {
	h := deliveredFixture(t, thisNodeFartherThanEveryPeer(), handoffRedundancy)

	responsiblePeers := []yacymodel.Hash{closerPeer()}
	if !h.handOff(t, fixtureWordPartition(), responsiblePeers, responsiblePeers) {
		t.Fatal(
			"word partition kept, want it handed off: redundancy is met by a closer responsible peer",
		)
	}
	if held := h.postingsOf(t, fixtureWordPartition()); len(held) != 0 {
		t.Fatalf("postings left after handoff = %v, want none", held)
	}
	if holders := h.holdersOf(t, fixtureWordPartition()); len(holders) != 0 {
		t.Fatalf("holders left after handoff = %v, want them forgotten", holders)
	}
}

func TestPostingsOfOtherWordPartitionsSurviveAHandoff(t *testing.T) {
	h := deliveredFixture(t, thisNodeFartherThanEveryPeer(), handoffRedundancy)
	h.admit(
		t,
		sameWordOtherPartition(),
		urlsIn(sameWordOtherPartition().Partition, postingsPerFixture)...)
	h.admit(
		t,
		otherWordSamePartition(),
		urlsIn(otherWordSamePartition().Partition, postingsPerFixture)...)
	h.deliver(t, sameWordOtherPartition(), otherWordSamePartition())

	responsiblePeers := []yacymodel.Hash{closerPeer()}
	h.handOff(t, fixtureWordPartition(), responsiblePeers, responsiblePeers)

	for _, wordPartition := range []yacymodel.WordPartition{
		sameWordOtherPartition(),
		otherWordSamePartition(),
	} {
		if held := h.postingsOf(t, wordPartition); len(held) != postingsPerFixture {
			t.Fatalf(
				"postings of %+v = %v, want all %d kept",
				wordPartition,
				held,
				postingsPerFixture,
			)
		}
	}
}

func TestWordPartitionKeptWhileOneOfItsPostingsIsUndelivered(t *testing.T) {
	h := deliveredFixture(t, thisNodeFartherThanEveryPeer(), handoffRedundancy)
	h.admit(
		t,
		fixtureWordPartition(),
		urlsIn(fixtureWordPartition().Partition, postingsPerFixture+1)...)

	responsiblePeers := []yacymodel.Hash{closerPeer()}
	if h.handOff(t, fixtureWordPartition(), responsiblePeers, responsiblePeers) {
		t.Fatal("word partition handed off, want it kept: one of its postings is undelivered")
	}
	assertKept(t, h, fixtureWordPartition(), postingsPerFixture+1)
}

func TestWordPartitionKeptBelowRedundancy(t *testing.T) {
	h := deliveredFixture(t, thisNodeFartherThanEveryPeer(), 2)

	responsiblePeers := []yacymodel.Hash{closerPeer(), otherCloserPeer()}
	if h.handOff(t, fixtureWordPartition(), []yacymodel.Hash{closerPeer()}, responsiblePeers) {
		t.Fatal("word partition handed off, want it kept: only one of two owed peers holds it")
	}
	assertKept(t, h, fixtureWordPartition(), postingsPerFixture)
}

func TestWordPartitionKeptWhileItsHoldersAreNotResponsiblePeers(t *testing.T) {
	h := deliveredFixture(t, thisNodeFartherThanEveryPeer(), handoffRedundancy)

	if h.handOff(
		t,
		fixtureWordPartition(),
		[]yacymodel.Hash{otherCloserPeer()},
		[]yacymodel.Hash{closerPeer()},
	) {
		t.Fatal("word partition handed off, want it kept: the holder is not a responsible peer")
	}
	assertKept(t, h, fixtureWordPartition(), postingsPerFixture)
}

func TestWordPartitionKeptWhileItsHolderIsNotCloserThanThisNode(t *testing.T) {
	h := deliveredFixture(t, thisNodeCloserThanEveryPeer(), handoffRedundancy)

	responsiblePeers := []yacymodel.Hash{closerPeer()}
	if h.handOff(t, fixtureWordPartition(), responsiblePeers, responsiblePeers) {
		t.Fatal("word partition handed off, want it kept: this node is closer than the holder")
	}
	assertKept(t, h, fixtureWordPartition(), postingsPerFixture)
}

func TestKeepEveryWordPartitionHandsOffNothing(t *testing.T) {
	responsiblePeers := []yacymodel.Hash{closerPeer()}
	wasHandedOff, err := wordpartitionhandoff.KeepEveryWordPartition{}.HandOffIfHeldByResponsiblePeers(
		context.Background(),
		nil,
		fixtureWordPartition(),
		responsiblePeers,
		responsiblePeers,
	)
	if err != nil {
		t.Fatalf("HandOffIfHeldByResponsiblePeers: %v", err)
	}
	if wasHandedOff {
		t.Fatal("word partition handed off, want every word partition kept")
	}
}
