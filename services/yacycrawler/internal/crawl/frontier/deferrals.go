package frontier

type deferredVisits []PendingVisit

func (d deferredVisits) Len() int { return len(d) }

func (d deferredVisits) Less(i, j int) bool { return d[i].notBefore.Before(d[j].notBefore) }

func (d deferredVisits) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d *deferredVisits) Push(visit any) { *d = append(*d, visit.(PendingVisit)) }

func (d *deferredVisits) Pop() any {
	old := *d
	last := len(old) - 1
	visit := old[last]
	*d = old[:last]
	return visit
}
