package wordpartitionholders_test

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
)

const (
	partitionExponent = 4
	amountOfWords     = 16
	refreshInterval   = time.Hour
)

type harness struct {
	vault      *vault.Vault
	partitions yacymodel.DHTRingPartitions
	holders    *wordpartitionholders.Holders
	clock      time.Time
}

func openHarness(t *testing.T) *harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	h := &harness{vault: v, partitions: partitions, clock: time.Unix(1_700_000_000, 0).UTC()}
	opened, err := wordpartitionholders.Open(v, partitions, refreshInterval, h.now)
	if err != nil {
		t.Fatalf("wordpartitionholders.Open: %v", err)
	}
	h.holders = opened

	return h
}

func (h *harness) now() time.Time {
	return h.clock
}

func (h *harness) advance(duration time.Duration) {
	h.clock = h.clock.Add(duration)
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

func (h *harness) forget(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.Forget(tx, wordPartition)
	})
}

func (h *harness) holdersOf(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
) []yacymodel.Hash {
	t.Helper()

	var holders []yacymodel.Hash
	h.read(t, func(tx *vault.Txn) error {
		var err error
		holders, err = h.holders.Of(tx, wordPartition)

		return err
	})

	return holders
}

func (h *harness) dueRefreshesIn(
	t *testing.T,
	sector yacymodel.DHTRingSector,
	limit int,
) []yacymodel.WordPartition {
	t.Helper()

	var dueWordPartitions []yacymodel.WordPartition
	h.read(t, func(tx *vault.Txn) error {
		var err error
		dueWordPartitions, err = h.holders.DueRefreshesIn(tx, sector, limit)

		return err
	})

	return dueWordPartitions
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

func (h *harness) sectorOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(
		yacymodel.DHTRingPositionOfWordPartition(wordPartition, h.partitions),
	)
}

func (h *harness) everyWordPartition() []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, amountOfWords*int(h.partitions))
	for word := range amountOfWords {
		for partition := range uint(h.partitions) {
			wordPartitions = append(wordPartitions, yacymodel.WordPartition{
				Word:      yacymodel.WordHash("word" + strconv.Itoa(word)),
				Partition: partition,
			})
		}
	}

	return wordPartitions
}

func (h *harness) wordPartitionsOfBusiestSector() []yacymodel.WordPartition {
	wordPartitionsPerSector := map[yacymodel.DHTRingSector][]yacymodel.WordPartition{}
	var busiest yacymodel.DHTRingSector
	for _, wordPartition := range h.everyWordPartition() {
		sector := h.sectorOf(wordPartition)
		wordPartitionsPerSector[sector] = append(wordPartitionsPerSector[sector], wordPartition)
		if len(wordPartitionsPerSector[sector]) > len(wordPartitionsPerSector[busiest]) {
			busiest = sector
		}
	}

	return wordPartitionsPerSector[busiest]
}

func peer(name string) yacymodel.Hash {
	return yacymodel.WordHash(name)
}

func someWordPartition() yacymodel.WordPartition {
	return yacymodel.WordPartition{Word: yacymodel.WordHash("word"), Partition: 3}
}

func TestRecordedHoldersReadBack(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	holders := []yacymodel.Hash{peer("peer0"), peer("peer1")}

	h.recordHolders(t, wordPartition, holders...)

	if got := h.holdersOf(t, wordPartition); !slices.Equal(got, holders) {
		t.Fatalf("Of = %v, want %v", got, holders)
	}
}

func TestAnUnknownWordPartitionHasNoHolders(t *testing.T) {
	h := openHarness(t)
	h.recordHolders(t, someWordPartition(), peer("peer0"))

	unknown := yacymodel.WordPartition{Word: someWordPartition().Word, Partition: 4}
	if got := h.holdersOf(t, unknown); got != nil {
		t.Fatalf("Of an unknown word partition = %v, want nil", got)
	}
}

func TestAWordPartitionIsDueOnlyOnceItsRefreshTimeHasPassed(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	sector := h.sectorOf(wordPartition)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.advance(refreshInterval - time.Second)
	if due := h.dueRefreshesIn(t, sector, 10); len(due) != 0 {
		t.Fatalf("due refreshes before the refresh time = %v, want none", due)
	}

	h.advance(time.Second)
	if due := h.dueRefreshesIn(t, sector, 10); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due refreshes at the refresh time = %v, want %v", due, wordPartition)
	}
}

func TestRecordingAgainMovesTheRefreshTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	sector := h.sectorOf(wordPartition)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.advance(refreshInterval / 2)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.advance(refreshInterval / 2)
	if due := h.dueRefreshesIn(t, sector, 10); len(due) != 0 {
		t.Fatalf("due refreshes at the earlier refresh time = %v, want none", due)
	}
	h.advance(refreshInterval / 2)
	if due := h.dueRefreshesIn(t, sector, 10); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due refreshes at the moved refresh time = %v, want %v once", due, wordPartition)
	}
}

func TestDueRefreshesComeOnlyFromTheirSector(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)

	seen := 0
	for sector := range yacymodel.DHTRingSectorCount {
		due := h.dueRefreshesIn(t, sector, len(h.everyWordPartition()))
		for _, wordPartition := range due {
			if h.sectorOf(wordPartition) != sector {
				t.Fatalf("due refreshes of sector %d hold %+v of sector %d",
					sector, wordPartition, h.sectorOf(wordPartition))
			}
		}
		seen += len(due)
	}
	if seen != len(h.everyWordPartition()) {
		t.Fatalf("due refreshes of all sectors = %d, want %d", seen, len(h.everyWordPartition()))
	}
}

func TestDueRefreshesComeOldestFirstUpToTheLimit(t *testing.T) {
	h := openHarness(t)
	wordPartitions := h.wordPartitionsOfBusiestSector()
	if len(wordPartitions) < 3 {
		t.Fatalf("busiest sector holds %d word partitions, want at least 3", len(wordPartitions))
	}
	sector := h.sectorOf(wordPartitions[0])
	oldestFirst := slices.Clone(wordPartitions)
	slices.Reverse(oldestFirst)
	for _, wordPartition := range oldestFirst {
		h.recordHolders(t, wordPartition, peer("peer0"))
		h.advance(time.Minute)
	}
	h.advance(refreshInterval)

	if due := h.dueRefreshesIn(t, sector, len(oldestFirst)); !slices.Equal(due, oldestFirst) {
		t.Fatalf("due refreshes = %v, want %v", due, oldestFirst)
	}
	if due := h.dueRefreshesIn(t, sector, 2); !slices.Equal(due, oldestFirst[:2]) {
		t.Fatalf("due refreshes with limit 2 = %v, want %v", due, oldestFirst[:2])
	}
}

func TestRecordingNoHoldersClearsTheWordPartition(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.recordHolders(t, wordPartition)

	h.assertCleared(t, wordPartition)
}

func TestForgetClearsTheWordPartition(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.forget(t, wordPartition)

	h.assertCleared(t, wordPartition)
}

func TestForgetOfAnUnknownWordPartitionKeepsTheOthers(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.forget(t, yacymodel.WordPartition{Word: wordPartition.Word, Partition: 4})

	if got := h.holdersOf(t, wordPartition); len(got) != 1 {
		t.Fatalf("Of after forgetting another word partition = %v, want one holder", got)
	}
}

func (h *harness) assertCleared(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	if got := h.holdersOf(t, wordPartition); got != nil {
		t.Fatalf("Of a cleared word partition = %v, want nil", got)
	}
	h.advance(refreshInterval)
	if due := h.dueRefreshesIn(t, h.sectorOf(wordPartition), 10); len(due) != 0 {
		t.Fatalf("due refreshes of a cleared word partition = %v, want none", due)
	}
}
