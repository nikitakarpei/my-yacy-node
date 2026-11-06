// Package servicehealth holds whether a service is healthy: it has started and each of its
// broker connections is up. The process that runs the service reads it or hears of each
// change.
package servicehealth

import "sync/atomic"

type Health struct {
	started         atomic.Bool
	lostConnections atomic.Int32
	changed         func()
}

func ReportingTo(changed func()) *Health {
	return &Health{changed: changed}
}

func (h *Health) MarkStarted() {
	h.started.Store(true)
	h.report()
}

func (h *Health) ConnectionLost() {
	h.lostConnections.Add(1)
	h.report()
}

func (h *Health) ConnectionRestored() {
	h.lostConnections.Add(-1)
	h.report()
}

func (h *Health) report() {
	if h.changed != nil {
		h.changed()
	}
}

func (h *Health) Started() bool {
	return h.started.Load()
}

func (h *Health) Connected() bool {
	return h.lostConnections.Load() <= 0
}

func (h *Health) Healthy() bool {
	return h.Started() && h.Connected()
}
