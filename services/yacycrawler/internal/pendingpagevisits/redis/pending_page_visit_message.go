package redis

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
)

type pendingPageVisitMessage struct {
	visits   *PendingPageVisits
	field    string
	identity string
	arrival  time.Time
	body     []byte
}

func (m pendingPageVisitMessage) Body() []byte {
	return m.body
}

func (m pendingPageVisitMessage) Identity() string {
	return m.identity
}

func (m pendingPageVisitMessage) Arrival() time.Time {
	return m.arrival
}

func (m pendingPageVisitMessage) Acknowledge(ctx context.Context) {
	m.visits.acknowledge(ctx, m.field, m.identity)
}

func (m pendingPageVisitMessage) Return(ctx context.Context) {
	m.ReturnAfter(ctx, pullintake.ReturnPause)
}

func (m pendingPageVisitMessage) ReturnAfter(ctx context.Context, delay time.Duration) {
	m.visits.returnAfter(ctx, m.field, m.identity, delay)
}
