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
	firstRetryDelay   = time.Minute
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
	opened, err := wordpartitionholders.Open(v, partitions, refreshInterval, firstRetryDelay, h.now)
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
	recordedHolders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.RecordAtRedundancy(tx, wordPartition, recordedHolders)
	})
}

func (h *harness) recordHoldersShortOfRedundancy(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	recordedHolders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.RecordShortOfRedundancy(tx, wordPartition, recordedHolders)
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

	var recordedHolders []yacymodel.Hash
	h.read(t, func(tx *vault.Txn) error {
		var err error
		recordedHolders, err = h.holders.Of(tx, wordPartition)

		return err
	})

	return recordedHolders
}

func (h *harness) isInRetry(t *testing.T, wordPartition yacymodel.WordPartition) bool {
	t.Helper()

	var isInRetry bool
	h.read(t, func(tx *vault.Txn) error {
		var err error
		isInRetry, err = h.holders.IsInRetry(tx, wordPartition)

		return err
	})

	return isInRetry
}

func (h *harness) dueRefreshesAfter(
	t *testing.T,
	lastReadRefresh wordpartitionholders.ScheduledRefresh,
	refreshesBudget int,
) ([]wordpartitionholders.ScheduledRefresh, bool) {
	t.Helper()

	var (
		dueRefreshes []wordpartitionholders.ScheduledRefresh
		reachedEnd   bool
	)
	h.read(t, func(tx *vault.Txn) error {
		var err error
		dueRefreshes, reachedEnd, err = h.holders.DueRefreshesAfter(
			tx, lastReadRefresh, refreshesBudget)

		return err
	})

	return dueRefreshes, reachedEnd
}

func (h *harness) dueWordPartitions(t *testing.T, refreshesBudget int) []yacymodel.WordPartition {
	t.Helper()

	dueRefreshes, _ := h.dueRefreshesAfter(
		t, wordpartitionholders.ScheduledRefresh{}, refreshesBudget)

	return wordPartitionsOf(dueRefreshes)
}

func wordPartitionsOf(
	dueRefreshes []wordpartitionholders.ScheduledRefresh,
) []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(dueRefreshes))
	for _, dueRefresh := range dueRefreshes {
		wordPartitions = append(wordPartitions, dueRefresh.WordPartition)
	}

	return wordPartitions
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

func TestAWordPartitionAtRedundancyIsDueAfterTheRefreshInterval(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()

	h.recordHolders(t, wordPartition, peer("peer0"))

	h.assertDueOnlyAfter(t, wordPartition, refreshInterval)
}

func (h *harness) assertDueOnlyAfter(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	delay time.Duration,
) {
	t.Helper()

	h.advance(delay - time.Second)
	if due := h.dueWordPartitions(t, 10); len(due) != 0 {
		t.Fatalf("due refreshes before %v = %v, want none", delay, due)
	}
	h.advance(time.Second)
	if due := h.dueWordPartitions(t, 10); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due refreshes after %v = %v, want %v", delay, due, wordPartition)
	}
}

func TestRecordingAgainMovesTheRefreshTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.advance(refreshInterval / 2)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.advance(refreshInterval / 2)
	if due := h.dueWordPartitions(t, 10); len(due) != 0 {
		t.Fatalf("due refreshes at the earlier refresh time = %v, want none", due)
	}
	h.advance(refreshInterval / 2)
	if due := h.dueWordPartitions(t, 10); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due refreshes at the moved refresh time = %v, want %v once", due, wordPartition)
	}
}

func TestDueRefreshesGoOnceAroundTheRingSectorBySector(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)

	due := h.dueWordPartitions(t, len(h.everyWordPartition())+1)

	if len(due) != len(h.everyWordPartition()) {
		t.Fatalf("due refreshes = %d, want %d", len(due), len(h.everyWordPartition()))
	}
	for _, wordPartition := range h.everyWordPartition() {
		if !slices.Contains(due, wordPartition) {
			t.Fatalf("due refreshes miss %+v", wordPartition)
		}
	}
	for index := 1; index < len(due); index++ {
		if h.sectorOf(due[index]) < h.sectorOf(due[index-1]) {
			t.Fatalf("due refreshes go back to an earlier sector at %d", index)
		}
	}
}

func TestADueRefreshTellsTheSectorOfItsWordPartition(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)

	dueRefreshes, _ := h.dueRefreshesAfter(
		t, wordpartitionholders.ScheduledRefresh{}, len(h.everyWordPartition()))

	for _, dueRefresh := range dueRefreshes {
		if got, want := dueRefresh.Sector(), h.sectorOf(dueRefresh.WordPartition); got != want {
			t.Fatalf("Sector of %+v = %d, want %d", dueRefresh.WordPartition, got, want)
		}
	}
}

func TestDueRefreshesComeOldestFirstUpToTheLimit(t *testing.T) {
	h := openHarness(t)
	wordPartitions := h.wordPartitionsOfBusiestSector()
	if len(wordPartitions) < 3 {
		t.Fatalf("busiest sector holds %d word partitions, want at least 3", len(wordPartitions))
	}
	oldestFirst := slices.Clone(wordPartitions)
	slices.Reverse(oldestFirst)
	for _, wordPartition := range oldestFirst {
		h.recordHolders(t, wordPartition, peer("peer0"))
		h.advance(time.Minute)
	}
	h.advance(refreshInterval)

	if due := h.dueWordPartitions(t, len(oldestFirst)); !slices.Equal(due, oldestFirst) {
		t.Fatalf("due refreshes = %v, want %v", due, oldestFirst)
	}
	if due := h.dueWordPartitions(t, 2); !slices.Equal(due, oldestFirst[:2]) {
		t.Fatalf("due refreshes with a budget of 2 = %v, want %v", due, oldestFirst[:2])
	}
}

func TestDueRefreshesResumeAfterTheLastReadOne(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
		h.advance(time.Second)
	}
	h.advance(refreshInterval)
	whole := h.dueWordPartitions(t, len(h.everyWordPartition()))

	head, _ := h.dueRefreshesAfter(t, wordpartitionholders.ScheduledRefresh{}, len(whole)/3)
	rest, _ := h.dueRefreshesAfter(t, head[len(head)-1], len(whole))

	if got := append(wordPartitionsOf(head), wordPartitionsOf(rest)...); !slices.Equal(
		got,
		whole,
	) {
		t.Fatalf("resumed due refreshes = %v, want %v", got, whole)
	}
}

func TestDueRefreshesTellWhetherTheyReachedTheEnd(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)
	amountOfRefreshes := len(h.everyWordPartition())

	head, headReachedEnd := h.dueRefreshesAfter(
		t, wordpartitionholders.ScheduledRefresh{}, amountOfRefreshes-1)
	rest, restReachedEnd := h.dueRefreshesAfter(t, head[len(head)-1], amountOfRefreshes)

	if headReachedEnd || len(head) != amountOfRefreshes-1 {
		t.Fatalf("first read = %d refreshes, end %v; want %d and no end",
			len(head), headReachedEnd, amountOfRefreshes-1)
	}
	if !restReachedEnd || len(rest) != 1 {
		t.Fatalf(
			"second read = %d refreshes, end %v; want 1 and the end",
			len(rest),
			restReachedEnd,
		)
	}
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
	if due := h.dueWordPartitions(t, 10); len(due) != 0 {
		t.Fatalf("due refreshes of a cleared word partition = %v, want none", due)
	}
}

func (h *harness) earliestRefreshAt(t *testing.T) (time.Time, bool) {
	t.Helper()

	var (
		earliestRefreshAt time.Time
		found             bool
	)
	h.read(t, func(tx *vault.Txn) error {
		var err error
		earliestRefreshAt, found, err = h.holders.EarliestRefreshAt(tx)

		return err
	})

	return earliestRefreshAt, found
}

func TestAWordPartitionShortOfRedundancyIsDueAfterTheRetryDelay(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()

	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))

	h.assertDueOnlyAfter(t, wordPartition, firstRetryDelay)
}

func TestAWordPartitionWithNoHoldersStaysDue(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.recordHoldersShortOfRedundancy(t, wordPartition)

	if got := h.holdersOf(t, wordPartition); len(got) != 0 {
		t.Fatalf("Of = %v, want no holders", got)
	}
	h.assertDueOnlyAfter(t, wordPartition, firstRetryDelay)
}

func TestNoRefreshIsEarliestWhileNoneIsScheduled(t *testing.T) {
	h := openHarness(t)

	if earliestRefreshAt, found := h.earliestRefreshAt(t); found {
		t.Fatalf("EarliestRefreshAt = %v, want none", earliestRefreshAt)
	}
}

func TestEarliestRefreshAtIsTheEarliestOverEverySector(t *testing.T) {
	h := openHarness(t)
	wordPartitions := h.everyWordPartition()
	for _, wordPartition := range wordPartitions[1:] {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	start := h.clock
	h.recordHoldersShortOfRedundancy(t, wordPartitions[0], peer("peer0"))

	if got, found := h.earliestRefreshAt(t); !found || !got.Equal(start.Add(firstRetryDelay)) {
		t.Fatalf("EarliestRefreshAt = %v, %v, want %v", got, found, start.Add(firstRetryDelay))
	}
}

func TestEachRetryShortOfRedundancyDoublesTheDelay(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()

	for _, delay := range []time.Duration{firstRetryDelay, 2 * firstRetryDelay, 4 * firstRetryDelay} {
		h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))

		h.assertDueOnlyAfter(t, wordPartition, delay)
	}
}

func TestTheRetryDelayStopsAtTheRefreshInterval(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	for delay := firstRetryDelay; delay < refreshInterval; delay *= 2 {
		h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))
	}

	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))

	h.assertDueOnlyAfter(t, wordPartition, refreshInterval)
}

func TestReachingRedundancyStartsTheRetryDelayOver(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))
	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))
	h.recordHolders(t, wordPartition, peer("peer0"), peer("peer1"))

	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))

	h.assertDueOnlyAfter(t, wordPartition, firstRetryDelay)
}

func TestAWordPartitionIsInRetryOnlyWhileShortOfRedundancy(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	if h.isInRetry(t, wordPartition) {
		t.Fatalf("IsInRetry = true, want false for an unknown word partition")
	}

	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))
	if !h.isInRetry(t, wordPartition) {
		t.Fatalf("IsInRetry = false, want true while short of redundancy")
	}

	h.recordHolders(t, wordPartition, peer("peer0"), peer("peer1"))
	if h.isInRetry(t, wordPartition) {
		t.Fatalf("IsInRetry = true, want false once at redundancy")
	}
}
