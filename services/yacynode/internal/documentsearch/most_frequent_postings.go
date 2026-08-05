package documentsearch

import "container/heap"

type postingsLeastFrequentFirst []termPosting

func (h postingsLeastFrequentFirst) Len() int { return len(h) }

func (h postingsLeastFrequentFirst) Less(i, j int) bool {
	return h[i].occurrences < h[j].occurrences
}

func (h postingsLeastFrequentFirst) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *postingsLeastFrequentFirst) Push(x any) { *h = append(*h, x.(termPosting)) }

func (h *postingsLeastFrequentFirst) Pop() any {
	postings := *h
	lastPosting := postings[len(postings)-1]
	*h = postings[:len(postings)-1]

	return lastPosting
}

type mostFrequentPostings struct {
	maxPostings int
	postings    postingsLeastFrequentFirst
}

func (m *mostFrequentPostings) consider(candidate termPosting) {
	if m.maxPostings <= 0 || len(m.postings) < m.maxPostings {
		heap.Push(&m.postings, candidate)

		return
	}
	if m.postings[0].occurrences < candidate.occurrences {
		m.postings[0] = candidate
		heap.Fix(&m.postings, 0)
	}
}
