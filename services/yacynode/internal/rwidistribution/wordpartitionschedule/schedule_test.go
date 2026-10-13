package wordpartitionschedule_test

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
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
	schedule   *wordpartitionschedule.Schedule
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
	opened, err := wordpartitionschedule.Open(
		context.Background(),
		v,
		partitions,
		refreshInterval,
		firstRetryDelay,
		h.now,
	)
	if err != nil {
		t.Fatalf("wordpartitionschedule.Open: %v", err)
	}
	h.schedule = opened

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
		return h.schedule.RecordRefreshed(tx, wordPartition, recordedHolders)
	})
}

func (h *harness) recordHoldersAtRedundancy(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	recordedHolders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.schedule.RecordAtRedundancy(tx, wordPartition, recordedHolders)
	})
}

func (h *harness) recordHoldersShortOfRedundancy(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	recordedHolders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.schedule.RecordShortOfRedundancy(tx, wordPartition, recordedHolders)
	})
}

func (h *harness) forget(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.schedule.Forget(tx, wordPartition)
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
		recordedHolders, err = h.schedule.HoldersOf(tx, wordPartition)

		return err
	})

	return recordedHolders
}

func (h *harness) isInRetry(t *testing.T, wordPartition yacymodel.WordPartition) bool {
	t.Helper()

	var isInRetry bool
	h.read(t, func(tx *vault.Txn) error {
		var err error
		isInRetry, err = h.schedule.IsInRetry(tx, wordPartition)

		return err
	})

	return isInRetry
}

var wholeRing = yacymodel.DHTRingSpan{First: 0, Last: yacymodel.MaxDHTRingPosition}

func (h *harness) dueWithin(
	t *testing.T,
	span yacymodel.DHTRingSpan,
	lastRead wordpartitionschedule.ScheduledWordPartition,
	budget int,
) ([]wordpartitionschedule.ScheduledWordPartition, bool) {
	t.Helper()

	var (
		due        []wordpartitionschedule.ScheduledWordPartition
		reachedEnd bool
	)
	h.read(t, func(tx *vault.Txn) error {
		var err error
		due, reachedEnd, err = h.schedule.WordPartitionsDueWithin(tx, span, lastRead, budget)

		return err
	})

	return due, reachedEnd
}

func (h *harness) dueWordPartitions(t *testing.T, budget int) []yacymodel.WordPartition {
	t.Helper()

	due, _ := h.dueWithin(t, wholeRing, wordpartitionschedule.ScheduledWordPartition{}, budget)

	return wordPartitionsOf(due)
}

func wordPartitionsOf(
	due []wordpartitionschedule.ScheduledWordPartition,
) []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(due))
	for _, scheduled := range due {
		wordPartitions = append(wordPartitions, scheduled.WordPartition)
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
	return yacymodel.DHTRingSectorOf(h.positionOf(wordPartition))
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

const ringSliceShift = 53

func (h *harness) positionOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOfWordPartition(wordPartition, h.partitions)
}

func (h *harness) ringSliceOf(wordPartition yacymodel.WordPartition) uint64 {
	return uint64(h.positionOf(wordPartition)) >> ringSliceShift
}

func (h *harness) wordPartitionsSharingARingSlice(
	t *testing.T,
	amount int,
) []yacymodel.WordPartition {
	t.Helper()

	sharing := map[uint64][]yacymodel.WordPartition{}
	for word := range 20_000 {
		wordPartition := yacymodel.WordPartition{
			Word: yacymodel.WordHash("slice" + strconv.Itoa(word)),
		}
		slice := h.ringSliceOf(wordPartition)
		sharing[slice] = append(sharing[slice], wordPartition)
		if len(sharing[slice]) == amount {
			return sharing[slice]
		}
	}
	t.Fatalf("no %d word partitions share a ring slice", amount)

	return nil
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
		t.Fatalf("due word partitions before %v = %v, want none", delay, due)
	}
	h.advance(time.Second)
	if due := h.dueWordPartitions(t, 10); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due word partitions after %v = %v, want %v", delay, due, wordPartition)
	}
}

func TestRecordingAgainMovesTheDueTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.advance(refreshInterval / 2)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.advance(refreshInterval / 2)
	if due := h.dueWordPartitions(t, 10); len(due) != 0 {
		t.Fatalf("due word partitions at the earlier refresh time = %v, want none", due)
	}
	h.advance(refreshInterval / 2)
	if due := h.dueWordPartitions(t, 10); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf(
			"due word partitions at the moved refresh time = %v, want %v once",
			due,
			wordPartition,
		)
	}
}

func TestDueWordPartitionsGoOnceAroundTheRingSliceBySlice(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)

	due := h.dueWordPartitions(t, len(h.everyWordPartition())+1)

	if len(due) != len(h.everyWordPartition()) {
		t.Fatalf("due word partitions = %d, want %d", len(due), len(h.everyWordPartition()))
	}
	for _, wordPartition := range h.everyWordPartition() {
		if !slices.Contains(due, wordPartition) {
			t.Fatalf("due word partitions miss %+v", wordPartition)
		}
	}
	for index := 1; index < len(due); index++ {
		if h.ringSliceOf(due[index]) < h.ringSliceOf(due[index-1]) {
			t.Fatalf("due word partitions go back to an earlier ring slice at %d", index)
		}
	}
}

func TestADueWordPartitionTellsTheSectorOfItsPosition(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)

	due, _ := h.dueWithin(
		t, wholeRing, wordpartitionschedule.ScheduledWordPartition{}, len(h.everyWordPartition()))

	for _, scheduled := range due {
		if got, want := scheduled.Sector(), h.sectorOf(scheduled.WordPartition); got != want {
			t.Fatalf("Sector of %+v = %d, want %d", scheduled.WordPartition, got, want)
		}
	}
}

func TestDueWordPartitionsWithinASpanComeOnlyFromThatSpan(t *testing.T) {
	h := openHarness(t)
	everyWordPartition := h.everyWordPartition()
	for _, wordPartition := range everyWordPartition {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)
	span := yacymodel.DHTRingSpan{
		First: yacymodel.MaxDHTRingPosition / 4,
		Last:  yacymodel.MaxDHTRingPosition / 2,
	}
	var want []yacymodel.WordPartition
	for _, wordPartition := range everyWordPartition {
		if span.Contains(h.positionOf(wordPartition)) {
			want = append(want, wordPartition)
		}
	}
	if len(want) == 0 || len(want) == len(everyWordPartition) {
		t.Fatalf("span holds %d of %d word partitions, want some but not all",
			len(want), len(everyWordPartition))
	}

	due, reachedEnd := h.dueWithin(
		t, span, wordpartitionschedule.ScheduledWordPartition{}, len(everyWordPartition))

	got := wordPartitionsOf(due)
	if len(got) != len(want) || !reachedEnd {
		t.Fatalf("due within the span = %d word partitions, end %v; want %d and the end",
			len(got), reachedEnd, len(want))
	}
	for _, wordPartition := range want {
		if !slices.Contains(got, wordPartition) {
			t.Fatalf("due within the span miss %+v", wordPartition)
		}
	}
}

func TestDueWordPartitionsInASpanEdgeSliceStopAtTheSpanEdge(t *testing.T) {
	h := openHarness(t)
	sharing := h.wordPartitionsSharingARingSlice(t, 3)
	for _, wordPartition := range sharing {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)
	positions := make([]yacymodel.DHTRingPosition, 0, len(sharing))
	for _, wordPartition := range sharing {
		positions = append(positions, h.positionOf(wordPartition))
	}
	slices.Sort(positions)
	span := yacymodel.DHTRingSpan{First: positions[1], Last: yacymodel.MaxDHTRingPosition}

	due, _ := h.dueWithin(t, span, wordpartitionschedule.ScheduledWordPartition{}, len(sharing))

	if len(due) != 2 {
		t.Fatalf("due within a span that starts inside a ring slice = %d, want 2", len(due))
	}
	for _, wordPartition := range wordPartitionsOf(due) {
		if h.positionOf(wordPartition) < span.First {
			t.Fatalf("due word partition %+v sits before the span", wordPartition)
		}
	}
}

func TestDueWordPartitionsComeOldestFirstInARingSliceUpToTheBudget(t *testing.T) {
	h := openHarness(t)
	oldestFirst := h.wordPartitionsSharingARingSlice(t, 3)
	for _, wordPartition := range oldestFirst {
		h.recordHolders(t, wordPartition, peer("peer0"))
		h.advance(time.Minute)
	}
	h.advance(refreshInterval)

	if due := h.dueWordPartitions(t, len(oldestFirst)); !slices.Equal(due, oldestFirst) {
		t.Fatalf("due word partitions = %v, want %v", due, oldestFirst)
	}
	if due := h.dueWordPartitions(t, 2); !slices.Equal(due, oldestFirst[:2]) {
		t.Fatalf("due word partitions with a budget of 2 = %v, want %v", due, oldestFirst[:2])
	}
}

func TestDueWordPartitionsResumeAfterTheLastReadOne(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
		h.advance(time.Second)
	}
	h.advance(refreshInterval)
	whole := h.dueWordPartitions(t, len(h.everyWordPartition()))

	head, _ := h.dueWithin(
		t,
		wholeRing,
		wordpartitionschedule.ScheduledWordPartition{},
		len(whole)/3,
	)
	rest, _ := h.dueWithin(t, wholeRing, head[len(head)-1], len(whole))

	if got := append(wordPartitionsOf(head), wordPartitionsOf(rest)...); !slices.Equal(
		got,
		whole,
	) {
		t.Fatalf("resumed due word partitions = %v, want %v", got, whole)
	}
}

func TestDueWordPartitionsResumeAcrossARingSliceBoundary(t *testing.T) {
	h := openHarness(t)
	byRingSlice := h.everyWordPartition()
	slices.SortStableFunc(byRingSlice, func(left, right yacymodel.WordPartition) int {
		return cmp.Compare(h.ringSliceOf(left), h.ringSliceOf(right))
	})
	inFirstSlice, inLaterSlice := byRingSlice[0], byRingSlice[len(byRingSlice)-1]
	if h.ringSliceOf(inFirstSlice) == h.ringSliceOf(inLaterSlice) {
		t.Fatalf("every word partition sits in one ring slice, want two")
	}
	h.recordHolders(t, inLaterSlice, peer("peer0"))
	h.advance(time.Minute)
	h.recordHolders(t, inFirstSlice, peer("peer0"))
	h.advance(refreshInterval)

	head, headReachedEnd := h.dueWithin(
		t,
		wholeRing,
		wordpartitionschedule.ScheduledWordPartition{},
		1,
	)
	rest, restReachedEnd := h.dueWithin(t, wholeRing, head[0], 1)

	if headReachedEnd ||
		!slices.Equal(wordPartitionsOf(head), []yacymodel.WordPartition{inFirstSlice}) {
		t.Fatalf(
			"first read = %v, end %v; want the word partition of the first ring slice and no end",
			wordPartitionsOf(head),
			headReachedEnd,
		)
	}
	if !restReachedEnd ||
		!slices.Equal(wordPartitionsOf(rest), []yacymodel.WordPartition{inLaterSlice}) {
		t.Fatalf(
			"second read = %v, end %v; want the word partition of the later ring slice and the end",
			wordPartitionsOf(rest),
			restReachedEnd,
		)
	}
}

func TestDueWordPartitionsTellWhetherTheyReachedTheEnd(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)
	amount := len(h.everyWordPartition())

	head, headReachedEnd := h.dueWithin(
		t, wholeRing, wordpartitionschedule.ScheduledWordPartition{}, amount-1)
	rest, restReachedEnd := h.dueWithin(t, wholeRing, head[len(head)-1], amount)

	if headReachedEnd || len(head) != amount-1 {
		t.Fatalf("first read = %d word partitions, end %v; want %d and no end",
			len(head), headReachedEnd, amount-1)
	}
	if !restReachedEnd || len(rest) != 1 {
		t.Fatalf(
			"second read = %d word partitions, end %v; want 1 and the end",
			len(rest),
			restReachedEnd,
		)
	}
}

func TestAWordPartitionRecordedWithALaterDueTimeIsNotReadBeforeItIsDue(t *testing.T) {
	h := openHarness(t)
	sooner, later := someWordPartition(), yacymodel.WordPartition{
		Word: someWordPartition().Word, Partition: 4,
	}
	h.recordHoldersShortOfRedundancy(t, sooner, peer("peer0"))
	h.recordHolders(t, later, peer("peer0"))

	h.advance(firstRetryDelay)

	if due := h.dueWordPartitions(t, 10); !slices.Equal(due, []yacymodel.WordPartition{sooner}) {
		t.Fatalf("due word partitions = %v, want only %v", due, sooner)
	}
	h.advance(refreshInterval)
	if due := h.dueWordPartitions(t, 10); len(due) != 2 {
		t.Fatalf("due word partitions after the refresh interval = %v, want both", due)
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
		t.Fatalf("due word partitions of a cleared word partition = %v, want none", due)
	}
}

func (h *harness) earliestDueAt(t *testing.T) (time.Time, bool) {
	t.Helper()

	var (
		earliestDueAt time.Time
		found         bool
	)
	h.read(t, func(tx *vault.Txn) error {
		var err error
		earliestDueAt, found, err = h.schedule.EarliestDueAt(tx)

		return err
	})

	return earliestDueAt, found
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

func TestNoDueTimeIsEarliestWhileNoneIsScheduled(t *testing.T) {
	h := openHarness(t)

	if earliestDueAt, found := h.earliestDueAt(t); found {
		t.Fatalf("EarliestDueAt = %v, want none", earliestDueAt)
	}
}

func TestEarliestDueAtIsTheEarliestOverEveryRingSlice(t *testing.T) {
	h := openHarness(t)
	wordPartitions := h.everyWordPartition()
	for _, wordPartition := range wordPartitions[1:] {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	start := h.clock
	h.recordHoldersShortOfRedundancy(t, wordPartitions[0], peer("peer0"))

	if got, found := h.earliestDueAt(t); !found || !got.Equal(start.Add(firstRetryDelay)) {
		t.Fatalf("EarliestDueAt = %v, %v, want %v", got, found, start.Add(firstRetryDelay))
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

func (h *harness) makeDue(t *testing.T, wordPartition yacymodel.WordPartition) (wasMadeDue bool) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		var err error
		wasMadeDue, err = h.schedule.MakeDue(tx, wordPartition)

		return err
	})

	return wasMadeDue
}

func (h *harness) assertMadeDue(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	wantMadeDue bool,
) {
	t.Helper()

	if got := h.makeDue(t, wordPartition); got != wantMadeDue {
		t.Fatalf("MakeDue = %v, want %v", got, wantMadeDue)
	}
}

func TestMakingANewWordPartitionDueReportsThatItWrote(t *testing.T) {
	h := openHarness(t)

	h.assertMadeDue(t, someWordPartition(), true)
}

func TestMakingAHeldWordPartitionDueReportsThatItWrote(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.assertMadeDue(t, wordPartition, true)
}

func TestMakingAnAlreadyDueWordPartitionDueReportsNoWrite(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.makeDue(t, wordPartition)

	h.assertMadeDue(t, wordPartition, false)
}

func TestMakingAWordPartitionInRetryDueReportsNoWrite(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))

	h.assertMadeDue(t, wordPartition, false)
}

func (h *harness) earliestDueWithin(
	t *testing.T,
	span yacymodel.DHTRingSpan,
) (time.Time, bool) {
	t.Helper()

	var (
		earliestDueAt time.Time
		found         bool
	)
	h.read(t, func(tx *vault.Txn) error {
		var err error
		earliestDueAt, found, err = h.schedule.EarliestDueWithin(tx, span)

		return err
	})

	return earliestDueAt, found
}

func (h *harness) assertEarliestDueWithin(
	t *testing.T,
	span yacymodel.DHTRingSpan,
	want time.Time,
) {
	t.Helper()

	if got, found := h.earliestDueWithin(t, span); !found || !got.Equal(want) {
		t.Fatalf("EarliestDueWithin = %v, %v, want %v", got, found, want)
	}
}

func (h *harness) sortedWordPartitionsSharingARingSlice(
	t *testing.T,
	amount int,
) []yacymodel.WordPartition {
	t.Helper()

	sharing := h.wordPartitionsSharingARingSlice(t, amount)
	slices.SortFunc(sharing, func(a, b yacymodel.WordPartition) int {
		return cmp.Compare(h.positionOf(a), h.positionOf(b))
	})

	return sharing
}

func TestEarliestDueWithinASpanInsideOneRingSliceIgnoresTheRestOfTheRing(t *testing.T) {
	h := openHarness(t)
	sharing := h.sortedWordPartitionsSharingARingSlice(t, 3)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	start := h.clock
	h.advance(time.Second)
	h.recordHoldersShortOfRedundancy(t, sharing[1], peer("peer0"))
	h.advance(time.Second)
	h.recordHoldersShortOfRedundancy(t, sharing[2], peer("peer0"))
	span := yacymodel.DHTRingSpan{First: h.positionOf(sharing[0]), Last: h.positionOf(sharing[2])}

	h.assertEarliestDueWithin(t, span, start.Add(time.Second+firstRetryDelay))
}

func TestEarliestDueWithinASpanAcrossRingSlicesComesFromThatSpan(t *testing.T) {
	h := openHarness(t)
	span := yacymodel.DHTRingSpan{
		First: yacymodel.MaxDHTRingPosition / 4,
		Last:  yacymodel.MaxDHTRingPosition / 2,
	}
	var inside, outside yacymodel.WordPartition
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
		if span.Contains(h.positionOf(wordPartition)) {
			inside = wordPartition
		} else {
			outside = wordPartition
		}
	}
	start := h.clock
	h.recordHoldersShortOfRedundancy(t, outside, peer("peer0"))
	h.advance(time.Second)
	h.recordHoldersShortOfRedundancy(t, inside, peer("peer0"))

	h.assertEarliestDueWithin(t, span, start.Add(time.Second+firstRetryDelay))
}

func TestEarliestDueWithinASpanIgnoresEntriesJustOutsideBothEdges(t *testing.T) {
	h := openHarness(t)
	sharing := h.sortedWordPartitionsSharingARingSlice(t, 3)
	start := h.clock
	h.recordHolders(t, sharing[1], peer("peer0"))
	h.recordHoldersShortOfRedundancy(t, sharing[0], peer("peer0"))
	h.recordHoldersShortOfRedundancy(t, sharing[2], peer("peer0"))
	middle := h.positionOf(sharing[1])

	h.assertEarliestDueWithin(
		t,
		yacymodel.DHTRingSpan{First: middle, Last: middle},
		start.Add(refreshInterval),
	)
}

func TestNoDueTimeIsEarliestWithinAnEmptySpan(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}

	for _, span := range []yacymodel.DHTRingSpan{
		{First: 10, Last: 5},
		{First: yacymodel.MaxDHTRingPosition / 2, Last: 0},
	} {
		if earliestDueAt, found := h.earliestDueWithin(t, span); found {
			t.Fatalf("EarliestDueWithin %+v = %v, want none", span, earliestDueAt)
		}
	}
}

func TestAnUnknownWordPartitionBecomesDueNowWithNoHolders(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()

	h.makeDue(t, wordPartition)

	if due := h.dueWordPartitions(
		t,
		10,
	); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due word partitions = %v, want %v", due, wordPartition)
	}
	if got := h.holdersOf(t, wordPartition); len(got) != 0 {
		t.Fatalf("Of = %v, want no holders", got)
	}
	if h.isInRetry(t, wordPartition) {
		t.Fatal("IsInRetry = true, want false")
	}
}

func TestAWordPartitionAtRedundancyBecomesDueNowAndKeepsItsHolders(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"), peer("peer1"))

	h.makeDue(t, wordPartition)

	if due := h.dueWordPartitions(
		t,
		10,
	); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due word partitions = %v, want %v", due, wordPartition)
	}
	if got := h.holdersOf(
		t,
		wordPartition,
	); !slices.Equal(
		got,
		[]yacymodel.Hash{peer("peer0"), peer("peer1")},
	) {
		t.Fatalf("Of = %v, want both holders", got)
	}
	if h.isInRetry(t, wordPartition) {
		t.Fatal("IsInRetry = true, want false")
	}
}

func TestAWordPartitionInRetryIsNotMadeDueBeforeItsRetryTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))

	h.makeDue(t, wordPartition)

	if !h.isInRetry(t, wordPartition) {
		t.Fatal("IsInRetry = false, want true")
	}
	h.assertDueOnlyAfter(t, wordPartition, firstRetryDelay)
}

func TestAWordPartitionAlreadyDueStaysDueOnceWithItsHolders(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))
	h.advance(refreshInterval)

	h.makeDue(t, wordPartition)
	h.makeDue(t, wordPartition)

	if due := h.dueWordPartitions(
		t,
		10,
	); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due word partitions = %v, want %v", due, wordPartition)
	}
	if got := h.holdersOf(t, wordPartition); !slices.Equal(got, []yacymodel.Hash{peer("peer0")}) {
		t.Fatalf("Of = %v, want the recorded holder", got)
	}
}

func TestAnUndeliveredSendAtRedundancyKeepsTheRefreshTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))
	h.advance(refreshInterval / 2)

	h.recordHoldersAtRedundancy(t, wordPartition, peer("peer0"))

	h.assertDueOnlyAfter(t, wordPartition, refreshInterval/2)
}

func TestAWordPartitionNeverRefreshedIsDueAtOnceAfterASendAtRedundancy(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.makeDue(t, wordPartition)

	h.recordHoldersAtRedundancy(t, wordPartition, peer("peer0"))

	if due := h.dueWordPartitions(t, 10); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due word partitions = %v, want %v at once", due, wordPartition)
	}
}

func TestShortOfRedundancyKeepsTheLastRefreshTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))
	h.advance(refreshInterval / 4)
	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))
	h.advance(refreshInterval / 4)

	h.recordHoldersAtRedundancy(t, wordPartition, peer("peer0"))

	h.assertDueOnlyAfter(t, wordPartition, refreshInterval/2)
}

func TestMakingAHeldWordPartitionDueKeepsItsRefreshTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))
	h.advance(refreshInterval / 2)
	h.makeDue(t, wordPartition)

	h.recordHoldersAtRedundancy(t, wordPartition, peer("peer0"))

	h.assertDueOnlyAfter(t, wordPartition, refreshInterval/2)
}
