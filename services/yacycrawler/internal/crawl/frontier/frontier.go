// Package frontier holds the URLs a crawl order still owes a visit, in the order they come due.
package frontier

import (
	"container/heap"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Admission interface {
	Admit(canonicalURL canonicalurl.CanonicalURL, depth int) bool
}

type Frontier struct {
	config    Config
	admission Admission
	ready     []PendingVisit
	deferred  deferredVisits
}

func New(
	admission Admission,
	canonicalSeeds []canonicalurl.CanonicalURL,
	config Config,
) *Frontier {
	f := &Frontier{config: config, admission: admission}
	for _, seed := range canonicalSeeds {
		f.Admit(seed, 0)
	}
	return f
}

func (f *Frontier) Admit(canonicalURL canonicalurl.CanonicalURL, depth int) bool {
	if !f.admission.Admit(canonicalURL, depth) {
		return false
	}
	f.ready = append(f.ready, PendingVisit{URL: canonicalURL, Depth: depth})
	return true
}

func (f *Frontier) Peek() (PendingVisit, bool) {
	if len(f.ready) == 0 {
		return PendingVisit{}, false
	}
	return f.ready[0], true
}

func (f *Frontier) Drop() {
	f.ready = f.ready[1:]
}

func (f *Frontier) Defer(visit PendingVisit, now time.Time, deferFor time.Duration) bool {
	if visit.deferrals >= f.config.MaxDeferralsPerURL {
		return false
	}
	visit.deferrals++
	visit.notBefore = now.Add(deferFor)
	heap.Push(&f.deferred, visit)
	return true
}

func (f *Frontier) Retry(visit PendingVisit, now time.Time) bool {
	if visit.attempts >= f.config.MaxAttemptsPerURL {
		return false
	}
	visit.attempts++
	visit.notBefore = now.Add(f.config.RetryDelay.Delay(visit.attempts))
	heap.Push(&f.deferred, visit)
	return true
}

func (f *Frontier) EarliestDue() (time.Time, bool) {
	if len(f.deferred) == 0 {
		return time.Time{}, false
	}
	return f.deferred[0].notBefore, true
}

func (f *Frontier) PromoteDue(now time.Time) {
	for len(f.deferred) > 0 && !f.deferred[0].notBefore.After(now) {
		f.ready = append(f.ready, heap.Pop(&f.deferred).(PendingVisit))
	}
}

func (f *Frontier) Empty() bool {
	return len(f.ready) == 0 && len(f.deferred) == 0
}

func (f *Frontier) DrainPending() []PendingVisit {
	pending := make([]PendingVisit, 0, len(f.ready)+len(f.deferred))
	pending = append(pending, f.ready...)
	pending = append(pending, f.deferred...)
	f.ready = nil
	f.deferred = nil
	return pending
}
