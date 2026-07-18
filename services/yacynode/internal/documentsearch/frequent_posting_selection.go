package documentsearch

import "container/heap"

type leastFrequentFirst []termPosting

func (h leastFrequentFirst) Len() int           { return len(h) }
func (h leastFrequentFirst) Less(i, j int) bool { return h[i].occurrences < h[j].occurrences }
func (h leastFrequentFirst) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *leastFrequentFirst) Push(x any)        { *h = append(*h, x.(termPosting)) }

func (h *leastFrequentFirst) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]

	return last
}

type mostFrequentPostings struct {
	limit int
	kept  leastFrequentFirst
}

func (m *mostFrequentPostings) consider(candidate termPosting) {
	if m.limit <= 0 || len(m.kept) < m.limit {
		heap.Push(&m.kept, candidate)

		return
	}
	if m.kept[0].occurrences < candidate.occurrences {
		m.kept[0] = candidate
		heap.Fix(&m.kept, 0)
	}
}

func (m *mostFrequentPostings) collected() []termPosting {
	return m.kept
}
