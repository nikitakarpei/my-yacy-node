package ordertraversal

import (
	"container/heap"
	"time"
)

type entry struct {
	URL       string
	Depth     int
	Deferrals int
	Attempts  int
	NotBefore time.Time
}

type frontier struct {
	admission *profileAdmission
	ready     []entry
	deferred  deferrals
}

func newFrontier(admission *profileAdmission) *frontier {
	return &frontier{admission: admission}
}

func (f *frontier) Admit(url string, depth int) bool {
	if !f.admission.Admit(url, depth) {
		return false
	}
	f.ready = append(f.ready, entry{URL: url, Depth: depth})
	return true
}

func (f *frontier) HasReady() bool {
	return len(f.ready) > 0
}

func (f *frontier) Peek() (entry, bool) {
	if len(f.ready) == 0 {
		return entry{}, false
	}
	return f.ready[0], true
}

func (f *frontier) Next() (entry, bool) {
	if len(f.ready) == 0 {
		return entry{}, false
	}
	next := f.ready[0]
	f.ready = f.ready[1:]
	return next, true
}

func (f *frontier) Defer(e entry) {
	heap.Push(&f.deferred, e)
}

func (f *frontier) EarliestDue() (time.Time, bool) {
	if len(f.deferred) == 0 {
		return time.Time{}, false
	}
	return f.deferred[0].NotBefore, true
}

func (f *frontier) PromoteDue(now time.Time) {
	for len(f.deferred) > 0 && !f.deferred[0].NotBefore.After(now) {
		f.ready = append(f.ready, heap.Pop(&f.deferred).(entry))
	}
}

func (f *frontier) Drained() bool {
	return len(f.ready) == 0 && len(f.deferred) == 0
}

func (f *frontier) DrainPending() int {
	pending := len(f.ready) + len(f.deferred)
	f.ready = nil
	f.deferred = nil
	return pending
}
