package pebblevault

import (
	"slices"
	"sync"
)

type commitNumber uint64

type writeTransactionsHistory struct {
	guard                         sync.Mutex
	lastCommit                    commitNumber
	commits                       []recordedCommit
	runningTransactionsBegunAfter map[commitNumber]int
}

type recordedCommit struct {
	number      commitNumber
	writtenKeys sortedKeys
}

func newWriteTransactionsHistory() *writeTransactionsHistory {
	return &writeTransactionsHistory{runningTransactionsBegunAfter: map[commitNumber]int{}}
}

func (h *writeTransactionsHistory) transactionBegan() commitNumber {
	h.guard.Lock()
	defer h.guard.Unlock()

	h.runningTransactionsBegunAfter[h.lastCommit]++

	return h.lastCommit
}

func (h *writeTransactionsHistory) transactionEnded(begunAfter commitNumber) {
	h.guard.Lock()
	defer h.guard.Unlock()

	h.forgetTransactionBegunAfter(begunAfter)
	h.forgetCommitsBeforeOldestTransaction()
}

func (h *writeTransactionsHistory) forgetTransactionBegunAfter(begunAfter commitNumber) {
	h.runningTransactionsBegunAfter[begunAfter]--
	if h.runningTransactionsBegunAfter[begunAfter] == 0 {
		delete(h.runningTransactionsBegunAfter, begunAfter)
	}
}

func (h *writeTransactionsHistory) forgetCommitsBeforeOldestTransaction() {
	oldestBegunAfter := h.lastCommit
	for begunAfter := range h.runningTransactionsBegunAfter {
		oldestBegunAfter = min(oldestBegunAfter, begunAfter)
	}
	h.commits = slices.DeleteFunc(h.commits, func(commit recordedCommit) bool {
		return commit.number <= oldestBegunAfter
	})
}

func (h *writeTransactionsHistory) keyWrittenAfter(
	begunAfter commitNumber,
	footprint *readFootprint,
) ([]byte, bool) {
	h.guard.Lock()
	defer h.guard.Unlock()

	for _, commit := range h.commits {
		if commit.number <= begunAfter {
			continue
		}
		if key, read := footprint.readKeyAmong(commit.writtenKeys); read {
			return key, true
		}
	}

	return nil, false
}

func (h *writeTransactionsHistory) transactionCommitted(writtenKeys sortedKeys) {
	h.guard.Lock()
	defer h.guard.Unlock()

	h.lastCommit++
	h.commits = append(h.commits, recordedCommit{number: h.lastCommit, writtenKeys: writtenKeys})
}
