package termpostings

import (
	"container/heap"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingsLeastFrequentFirst []yacymodel.RWIPosting

func (h postingsLeastFrequentFirst) Len() int { return len(h) }

func (h postingsLeastFrequentFirst) Less(i, j int) bool {
	return h[i].Hits < h[j].Hits
}

func (h postingsLeastFrequentFirst) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *postingsLeastFrequentFirst) Push(x any) {
	*h = append(*h, x.(yacymodel.RWIPosting))
}

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

func (m *mostFrequentPostings) consider(candidate yacymodel.RWIPosting) {
	if m.maxPostings <= 0 || len(m.postings) < m.maxPostings {
		heap.Push(&m.postings, candidate)

		return
	}
	if m.postings[0].Hits < candidate.Hits {
		m.postings[0] = candidate
		heap.Fix(&m.postings, 0)
	}
}
