package wordpartitionschedule_test

import (
	"cmp"
	"context"
	"math"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
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
	writer     rwipostings.PostingWriter
	clock      time.Time
}

func openHarness(t *testing.T) *harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil, wallclock.Clock{})
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
	h.writer = opened.WriterFrom(&fakeWriter{}, &recordingAdmissionObserver{})

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

func (h *harness) dueWordPartitions(t *testing.T) []yacymodel.WordPartition {
	t.Helper()

	due, _ := h.dueWithin(t, wholeRing, wordpartitionschedule.ScheduledWordPartition{}, math.MaxInt)

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

func (h *harness) makeDue(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.admit(t, wordPartition)
	h.scheduleAdmissionsWithin(t, wholeRing)
}

func (h *harness) admit(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.writer.Write(tx, h.postingIn(wordPartition))
	})
}

func (h *harness) postingIn(wordPartition yacymodel.WordPartition) yacymodel.RWIPosting {
	for document := 0; ; document++ {
		url, err := yacymodel.ParseURLHash(
			yacymodel.WordHash("url" + strconv.Itoa(document)).String(),
		)
		if err == nil && h.partitions.PartitionOf(url) == wordPartition.Partition {
			return yacymodel.RWIPosting{WordHash: wordPartition.Word, URLHash: url}
		}
	}
}

func (h *harness) scheduleAdmissionsWithin(t *testing.T, spans ...yacymodel.DHTRingSpan) {
	t.Helper()

	if err := h.schedule.ScheduleAdmissionsWithin(context.Background(), spans); err != nil {
		t.Fatalf("ScheduleAdmissionsWithin: %v", err)
	}
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

func (h *harness) positionOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOfWordPartition(wordPartition, h.partitions)
}

func (h *harness) closestNeighbours(amount int) []yacymodel.WordPartition {
	candidates := make([]yacymodel.WordPartition, 0, 20_000)
	for word := range cap(candidates) {
		candidates = append(candidates, yacymodel.WordPartition{
			Word: yacymodel.WordHash("neighbour" + strconv.Itoa(word)),
		})
	}
	slices.SortFunc(candidates, func(a, b yacymodel.WordPartition) int {
		return cmp.Compare(h.positionOf(a), h.positionOf(b))
	})
	closest := 0
	for first := range len(candidates) - amount + 1 {
		if h.stretchOf(candidates[first:first+amount]) <
			h.stretchOf(candidates[closest:closest+amount]) {
			closest = first
		}
	}

	return candidates[closest : closest+amount]
}

func (h *harness) stretchOf(
	sortedWordPartitions []yacymodel.WordPartition,
) yacymodel.DHTRingPosition {
	return h.positionOf(sortedWordPartitions[len(sortedWordPartitions)-1]) -
		h.positionOf(sortedWordPartitions[0])
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
	if due := h.dueWordPartitions(t); len(due) != 0 {
		t.Fatalf("due word partitions before %v = %v, want none", delay, due)
	}
	h.advance(time.Second)
	if due := h.dueWordPartitions(t); !slices.Equal(
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
	if due := h.dueWordPartitions(t); len(due) != 0 {
		t.Fatalf("due word partitions at the earlier refresh time = %v, want none", due)
	}
	h.advance(refreshInterval / 2)
	if due := h.dueWordPartitions(t); !slices.Equal(
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

func TestResumedReadsReturnEachDueWordPartitionOnce(t *testing.T) {
	h := openHarness(t)
	everyWordPartition := h.everyWordPartition()
	for _, wordPartition := range everyWordPartition {
		h.recordHolders(t, wordPartition, peer("peer0"))
		h.advance(time.Second)
	}
	h.advance(refreshInterval)

	readsPerWordPartition := map[yacymodel.WordPartition]int{}
	var lastRead wordpartitionschedule.ScheduledWordPartition
	for reachedEnd := false; !reachedEnd; {
		var due []wordpartitionschedule.ScheduledWordPartition
		due, reachedEnd = h.dueWithin(t, wholeRing, lastRead, 7)
		if !reachedEnd && len(due) == 0 {
			t.Fatal("a read before the end returned nothing")
		}
		for _, wordPartition := range wordPartitionsOf(due) {
			readsPerWordPartition[wordPartition]++
		}
		if len(due) > 0 {
			lastRead = due[len(due)-1]
		}
	}

	if len(readsPerWordPartition) != len(everyWordPartition) {
		t.Fatalf("resumed reads returned %d word partitions, want %d",
			len(readsPerWordPartition), len(everyWordPartition))
	}
	for wordPartition, amountOfReads := range readsPerWordPartition {
		if amountOfReads != 1 {
			t.Fatalf("resumed reads returned %+v %d times, want once", wordPartition, amountOfReads)
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

func TestDueWordPartitionsWithinASpanStartingBetweenNeighboursStopAtTheSpanEdge(t *testing.T) {
	h := openHarness(t)
	neighbours := h.closestNeighbours(3)
	for _, wordPartition := range neighbours {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)
	span := yacymodel.DHTRingSpan{
		First: h.positionOf(neighbours[1]),
		Last:  yacymodel.MaxDHTRingPosition,
	}

	due, _ := h.dueWithin(t, span, wordpartitionschedule.ScheduledWordPartition{}, len(neighbours))

	if got := wordPartitionsOf(due); len(got) != 2 || slices.Contains(got, neighbours[0]) {
		t.Fatalf("due within a span that starts between neighbours = %v, want %v",
			got, neighbours[1:])
	}
}

func TestAReadOfDueWordPartitionsStopsAtItsBudget(t *testing.T) {
	h := openHarness(t)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	h.advance(refreshInterval)

	due, reachedEnd := h.dueWithin(t, wholeRing, wordpartitionschedule.ScheduledWordPartition{}, 2)

	if len(due) != 2 || reachedEnd {
		t.Fatalf("read with a budget of 2 = %d word partitions, end %v; want 2 and no end",
			len(due), reachedEnd)
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
	if due := h.dueWordPartitions(t); len(due) != 0 {
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
		earliestDueAt, found, err = h.schedule.EarliestDueWithin(tx, wholeRing)

		return err
	})

	return earliestDueAt, found
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

func TestNoDueTimeIsEarliestOnTheWholeRingWhileNoneIsScheduled(t *testing.T) {
	h := openHarness(t)

	if earliestDueAt, found := h.earliestDueAt(t); found {
		t.Fatalf("EarliestDueWithin = %v, want none", earliestDueAt)
	}
}

func TestTheEarliestDueTimeOnTheWholeRingIsTheEarliestOverEveryRingSlice(t *testing.T) {
	h := openHarness(t)
	wordPartitions := h.everyWordPartition()
	for _, wordPartition := range wordPartitions[1:] {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	start := h.clock
	h.recordHoldersShortOfRedundancy(t, wordPartitions[0], peer("peer0"))

	if got, found := h.earliestDueAt(t); !found || !got.Equal(start.Add(firstRetryDelay)) {
		t.Fatalf("EarliestDueWithin = %v, %v, want %v", got, found, start.Add(firstRetryDelay))
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

func TestEarliestDueWithinASpanOfNeighboursIgnoresTheRestOfTheRing(t *testing.T) {
	h := openHarness(t)
	neighbours := h.closestNeighbours(3)
	for _, wordPartition := range h.everyWordPartition() {
		h.recordHolders(t, wordPartition, peer("peer0"))
	}
	start := h.clock
	h.advance(time.Second)
	h.recordHoldersShortOfRedundancy(t, neighbours[1], peer("peer0"))
	h.advance(time.Second)
	h.recordHoldersShortOfRedundancy(t, neighbours[2], peer("peer0"))
	span := yacymodel.DHTRingSpan{
		First: h.positionOf(neighbours[0]),
		Last:  h.positionOf(neighbours[2]),
	}

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
	neighbours := h.closestNeighbours(3)
	start := h.clock
	h.recordHolders(t, neighbours[1], peer("peer0"))
	h.recordHoldersShortOfRedundancy(t, neighbours[0], peer("peer0"))
	h.recordHoldersShortOfRedundancy(t, neighbours[2], peer("peer0"))
	middle := h.positionOf(neighbours[1])

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

	if due := h.dueWordPartitions(t); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due word partitions = %v, want %v", due, wordPartition)
	}
	if got := h.holdersOf(t, wordPartition); len(got) != 0 {
		t.Fatalf("Of = %v, want no holders", got)
	}
}

func TestAWordPartitionAtRedundancyBecomesDueNowAndKeepsItsHolders(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"), peer("peer1"))

	h.makeDue(t, wordPartition)

	if due := h.dueWordPartitions(t); !slices.Equal(
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
}

func TestAWordPartitionInRetryIsNotMadeDueBeforeItsRetryTime(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))

	h.makeDue(t, wordPartition)

	h.assertDueOnlyAfter(t, wordPartition, firstRetryDelay)
}

func TestAWordPartitionAlreadyDueStaysDueOnceWithItsHolders(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHolders(t, wordPartition, peer("peer0"))
	h.advance(refreshInterval)

	h.makeDue(t, wordPartition)
	h.makeDue(t, wordPartition)

	if due := h.dueWordPartitions(t); !slices.Equal(
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

	if due := h.dueWordPartitions(t); !slices.Equal(
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

func (h *harness) isRefreshDue(t *testing.T, wordPartition yacymodel.WordPartition) bool {
	t.Helper()

	var isRefreshDue bool
	h.read(t, func(tx *vault.Txn) error {
		var err error
		isRefreshDue, err = h.schedule.IsRefreshDue(tx, wordPartition)

		return err
	})

	return isRefreshDue
}

func TestARefreshIsDueWithoutAnEntryAndOnceTheRefreshIntervalPassed(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	if !h.isRefreshDue(t, wordPartition) {
		t.Fatal("IsRefreshDue = false, want true without an entry")
	}

	h.recordHolders(t, wordPartition, peer("peer0"))
	if h.isRefreshDue(t, wordPartition) {
		t.Fatal("IsRefreshDue = true, want false right after the refresh")
	}

	h.advance(refreshInterval)
	if !h.isRefreshDue(t, wordPartition) {
		t.Fatal("IsRefreshDue = false, want true after the refresh interval")
	}
}

func TestAnAdmissionAloneMakesNothingDue(t *testing.T) {
	h := openHarness(t)

	h.admit(t, someWordPartition())

	if due := h.dueWordPartitions(t); len(due) != 0 {
		t.Fatalf("due word partitions = %v, want none", due)
	}
}

func TestSchedulingAdmissionsWithinASpanLeavesTheRestOfTheRingForLater(t *testing.T) {
	h := openHarness(t)
	neighbours := h.closestNeighbours(2)
	inside, outside := neighbours[0], neighbours[1]
	h.admit(t, inside)
	h.admit(t, outside)

	h.scheduleAdmissionsWithin(t, h.spanOf(inside))

	if due := h.dueWordPartitions(t); !slices.Equal(due, []yacymodel.WordPartition{inside}) {
		t.Fatalf("due word partitions = %v, want only %v", due, inside)
	}
	h.scheduleAdmissionsWithin(t, wholeRing)
	if due := h.dueWordPartitions(t); len(due) != 2 {
		t.Fatalf("due word partitions = %v, want both", due)
	}
}

func (h *harness) spanOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingSpan {
	position := h.positionOf(wordPartition)

	return yacymodel.DHTRingSpan{First: position, Last: position}
}

func TestScheduledAdmissionsDoNotMakeARefreshedWordPartitionDueAgain(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.makeDue(t, wordPartition)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.scheduleAdmissionsWithin(t, wholeRing)

	h.assertDueOnlyAfter(t, wordPartition, refreshInterval)
}

func TestAnAdmissionAfterARefreshMakesTheWordPartitionDueAgain(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.admit(t, wordPartition)
	h.scheduleAdmissionsWithin(t, wholeRing)
	h.admit(t, wordPartition)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.scheduleAdmissionsWithin(t, wholeRing)

	if due := h.dueWordPartitions(t); !slices.Equal(
		due,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("due word partitions = %v, want %v", due, wordPartition)
	}
}

func TestAdmissionsDuringARetryAreNotScheduledAfterIt(t *testing.T) {
	h := openHarness(t)
	wordPartition := someWordPartition()
	h.recordHoldersShortOfRedundancy(t, wordPartition, peer("peer0"))
	h.makeDue(t, wordPartition)
	h.advance(firstRetryDelay)
	h.recordHolders(t, wordPartition, peer("peer0"))

	h.scheduleAdmissionsWithin(t, wholeRing)

	h.assertDueOnlyAfter(t, wordPartition, refreshInterval)
}
