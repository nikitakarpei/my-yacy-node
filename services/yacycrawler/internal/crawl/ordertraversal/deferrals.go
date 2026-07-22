package ordertraversal

type deferrals []entry

func (d deferrals) Len() int { return len(d) }

func (d deferrals) Less(i, j int) bool { return d[i].NotBefore.Before(d[j].NotBefore) }

func (d deferrals) Swap(i, j int) { d[i], d[j] = d[j], d[i] }

func (d *deferrals) Push(x any) { *d = append(*d, x.(entry)) }

func (d *deferrals) Pop() any {
	old := *d
	last := len(old) - 1
	e := old[last]
	*d = old[:last]
	return e
}
