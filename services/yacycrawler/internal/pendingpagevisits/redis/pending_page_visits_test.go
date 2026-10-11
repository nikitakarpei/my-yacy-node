package redis_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingpagevisits/redis"
)

const (
	lease         = time.Minute
	pageVisitsKey = "{YACY_CRAWL_FRONTIER}:page-visits"
)

var errNoDueVisit = errors.New("no page visit is due")

type clock struct {
	now time.Time
}

func (c *clock) Now() time.Time {
	return c.now
}

func (c *clock) Sleep(context.Context, time.Duration) error {
	return errNoDueVisit
}

func (c *clock) pass(d time.Duration) {
	c.now = c.now.Add(d)
}

type recordedFailures struct {
	leases          []error
	counts          []error
	acknowledgments []string
	returns         []string
}

func (r *recordedFailures) FrontierLeaseFailed(_ context.Context, err error) {
	r.leases = append(r.leases, err)
}

func (r *recordedFailures) FrontierCountFailed(_ context.Context, err error) {
	r.counts = append(r.counts, err)
}

func (r *recordedFailures) FrontierAcknowledgeFailed(
	_ context.Context,
	identity string,
	_ error,
) {
	r.acknowledgments = append(r.acknowledgments, identity)
}

func (r *recordedFailures) FrontierReturnFailed(
	_ context.Context,
	identity string,
	_ time.Duration,
	_ error,
) {
	r.returns = append(r.returns, identity)
}

type frontier struct {
	server   *miniredis.Miniredis
	clock    *clock
	failures *recordedFailures
	visits   *redis.PendingPageVisits
}

func newFrontier(t *testing.T) frontier {
	t.Helper()
	server := miniredis.RunT(t)
	visitClock := &clock{now: time.Unix(1_000_000, 0)}
	failures := &recordedFailures{}
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	return frontier{
		server:   server,
		clock:    visitClock,
		failures: failures,
		visits:   redis.New(client, lease, visitClock, failures),
	}
}

func pageVisitTo(t *testing.T, rawURL string) pendingpagevisit.PendingPageVisit {
	t.Helper()
	return pendingpagevisit.PendingPageVisit{
		OrderID: "o1",
		URL:     canonicalurltest.CanonicalURLOf(t, rawURL),
		Depth:   1,
	}
}

func (f frontier) publish(t *testing.T, pageVisit pendingpagevisit.PendingPageVisit) {
	t.Helper()
	if err := f.visits.Publish(t.Context(), pageVisit); err != nil {
		t.Fatalf("Publish: %v", err)
	}
}

func (f frontier) next(t *testing.T) pullintake.PendingMessage {
	t.Helper()
	message, err := f.visits.Next(t.Context())
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	return message
}

func (f frontier) requireNoDueVisit(t *testing.T) {
	t.Helper()
	if message, err := f.visits.Next(t.Context()); !errors.Is(err, errNoDueVisit) {
		t.Fatalf("Next = %v, %v; want no due page visit", message, err)
	}
}

func pageVisitOf(
	t *testing.T,
	message pullintake.PendingMessage,
) pendingpagevisit.PendingPageVisit {
	t.Helper()
	pageVisit, err := pendingpagevisit.UnmarshalPendingPageVisit(message.Body())
	if err != nil {
		t.Fatalf("UnmarshalPendingPageVisit: %v", err)
	}
	return pageVisit
}

func TestAPublishedPageVisitIsTakenWithItsArrival(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.publish(t, pageVisit)
	arrival := f.clock.now

	message := f.next(t)

	if got := pageVisitOf(t, message); got != pageVisit {
		t.Fatalf("page visit = %+v, want %+v", got, pageVisit)
	}
	if !message.Arrival().Equal(arrival) {
		t.Fatalf("Arrival = %v, want %v", message.Arrival(), arrival)
	}
}

func TestAPageVisitPublishedTwiceWhilePendingIsTakenOnce(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.publish(t, pageVisit)
	f.publish(t, pageVisit)

	f.next(t)

	f.requireNoDueVisit(t)
}

func TestATakenPageVisitComesBackOnlyOnceItsLeaseEnds(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	taken := f.next(t)

	f.clock.pass(lease - time.Millisecond)
	f.requireNoDueVisit(t)
	f.clock.pass(time.Millisecond)

	if again := f.next(t); again.Identity() != taken.Identity() {
		t.Fatalf("Identity = %q after the lease, want %q", again.Identity(), taken.Identity())
	}
}

func TestAnAcknowledgedPageVisitNeverComesBack(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))

	f.next(t).Acknowledge(t.Context())

	f.clock.pass(lease)
	f.requireNoDueVisit(t)
}

func TestAPageVisitPublishedAgainAfterItsAcknowledgmentCarriesANewIdentity(t *testing.T) {
	f := newFrontier(t)
	pageVisit := pageVisitTo(t, "http://host/page")
	f.publish(t, pageVisit)
	first := f.next(t)
	first.Acknowledge(t.Context())

	f.publish(t, pageVisit)

	if again := f.next(t); again.Identity() == first.Identity() {
		t.Fatalf("Identity = %q again, want a new identity", again.Identity())
	}
}

func TestAPageVisitReturnedAfterADelayIsDueOnlyOnceTheDelayPasses(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	delay := 3 * lease

	f.next(t).ReturnAfter(t.Context(), delay)

	f.clock.pass(delay - time.Millisecond)
	f.requireNoDueVisit(t)
	f.clock.pass(time.Millisecond)
	f.next(t)
}

func TestAReturnedPageVisitComesBackLater(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))

	f.next(t).Return(t.Context())

	f.requireNoDueVisit(t)
	f.clock.pass(time.Hour)
	f.next(t)
}

func TestPageVisitsAreTakenInTheOrderTheyFallDue(t *testing.T) {
	f := newFrontier(t)
	first, second := pageVisitTo(t, "http://host/first"), pageVisitTo(t, "http://host/second")
	f.publish(t, first)
	f.clock.pass(time.Millisecond)
	f.publish(t, second)

	f.next(t).ReturnAfter(t.Context(), time.Second)
	f.clock.pass(time.Second)

	if got := pageVisitOf(t, f.next(t)); got != second {
		t.Fatalf("page visit = %+v, want the one due first %+v", got, second)
	}
}

func TestAPublishTheServerRefusesFails(t *testing.T) {
	f := newFrontier(t)
	f.server.SetError("server down")

	if err := f.visits.Publish(t.Context(), pageVisitTo(t, "http://host/page")); err == nil {
		t.Fatal("Publish succeeded on a refusing server")
	}
}

func TestALeaseTheServerRefusesIsReportedAndNothingIsTaken(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	f.server.SetError("server down")

	f.requireNoDueVisit(t)

	if len(f.failures.leases) != 1 {
		t.Fatalf("lease failures = %v, want one", f.failures.leases)
	}
}

func TestAnUnreadablePageVisitIsReportedAndNothingIsTaken(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	fields, err := f.server.HKeys(pageVisitsKey)
	if err != nil {
		t.Fatalf("HKeys: %v", err)
	}
	for _, field := range fields {
		f.server.HSet(pageVisitsKey, field, "1 not-an-arrival {}")
	}

	f.requireNoDueVisit(t)

	if len(f.failures.leases) != 1 {
		t.Fatalf("lease failures = %v, want one", f.failures.leases)
	}
}

func TestSettlementsTheServerRefusesAreReported(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/page"))
	message := f.next(t)
	f.server.SetError("server down")

	message.Acknowledge(t.Context())
	message.ReturnAfter(t.Context(), time.Second)

	identity := message.Identity()
	if len(f.failures.acknowledgments) != 1 || f.failures.acknowledgments[0] != identity ||
		len(f.failures.returns) != 1 || f.failures.returns[0] != identity {
		t.Fatalf(
			"failures = %+v, want one acknowledgment and one return of %q",
			f.failures,
			identity,
		)
	}
}

func TestTheAmountOfPendingPageVisitsCountsTakenOnesUntilTheyAreAcknowledged(t *testing.T) {
	f := newFrontier(t)
	f.publish(t, pageVisitTo(t, "http://host/first"))
	f.publish(t, pageVisitTo(t, "http://host/second"))
	f.next(t).Acknowledge(t.Context())
	f.next(t)

	if amount, counted := f.visits.AmountOfPendingPageVisits(t.Context()); !counted || amount != 1 {
		t.Fatalf("AmountOfPendingPageVisits = %d, %v; want 1, true", amount, counted)
	}
}

func TestACountTheServerRefusesIsReported(t *testing.T) {
	f := newFrontier(t)
	f.server.SetError("server down")

	if _, counted := f.visits.AmountOfPendingPageVisits(t.Context()); counted {
		t.Fatal("AmountOfPendingPageVisits counted on a refusing server")
	}
	if len(f.failures.counts) != 1 {
		t.Fatalf("count failures = %v, want one", f.failures.counts)
	}
}
