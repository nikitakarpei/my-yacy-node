package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pagevisitrecords/redis"
)

const retention = time.Hour

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func pageVisitRecordsOn(
	server *miniredis.Miniredis,
	clock redis.Clock,
	observer redis.PageVisitRecordObserver,
) *redis.PageVisitRecords {
	client := goredis.NewClient(&goredis.Options{Addr: server.Addr()})
	return redis.New(client, retention, clock, observer)
}

func TestNoPageVisitOfAPageNeverVisited(t *testing.T) {
	records := pageVisitRecordsOn(miniredis.RunT(t), &manualClock{}, silentPageVisitRecords{})

	visit, visited := records.LastPageVisitOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)
	if visited {
		t.Fatalf("want no page visit, got %+v", visit)
	}
}

func TestARecordedPageVisitKeepsItsTimeAndPageVersion(t *testing.T) {
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{now: now}
	records := pageVisitRecordsOn(miniredis.RunT(t), clock, silentPageVisitRecords{})

	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	version := pagefetch.PageVersion{EntityTag: `"abc"`, ModifiedAt: now.Add(-time.Minute)}
	records.RecordPageVisit(context.Background(), url, version)

	clock.now = now.Add(30 * time.Minute)
	visit, visited := records.LastPageVisitOf(context.Background(), url)
	if !visited || !visit.VisitedAt.Equal(now) {
		t.Fatalf("visited at %v, want the page visit recorded at %v", visit.VisitedAt, now)
	}
	if visit.Version.EntityTag != version.EntityTag ||
		!visit.Version.ModifiedAt.Equal(version.ModifiedAt) {
		t.Fatalf("page version = %+v, want %+v", visit.Version, version)
	}
}

type silentPageVisitRecords struct{}

func (silentPageVisitRecords) PageVisitNotRecorded(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
}

func (silentPageVisitRecords) LastPageVisitNotRead(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
}

type recordedFailures struct{ causes []error }

func (f *recordedFailures) PageVisitNotRecorded(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	f.causes = append(f.causes, cause)
}

func (f *recordedFailures) LastPageVisitNotRead(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	f.causes = append(f.causes, cause)
}

func refusingServer(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	server := miniredis.RunT(t)
	server.SetError("server down")
	return server
}

func TestALastPageVisitTheServerRefusesIsReportedAsNeverVisited(t *testing.T) {
	failures := &recordedFailures{}
	records := pageVisitRecordsOn(refusingServer(t), &manualClock{}, failures)

	visit, visited := records.LastPageVisitOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)

	if visited {
		t.Fatalf("want no page visit, got %+v", visit)
	}
	if len(failures.causes) != 1 {
		t.Fatalf("reported %v, want the refusal of the server", failures.causes)
	}
}

func TestAPageVisitTheServerRefusesIsReported(t *testing.T) {
	failures := &recordedFailures{}
	records := pageVisitRecordsOn(refusingServer(t), &manualClock{}, failures)

	records.RecordPageVisit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
		pagefetch.PageVersion{},
	)

	if len(failures.causes) != 1 {
		t.Fatalf("reported %v, want the refusal of the server", failures.causes)
	}
}

func TestAPageVisitIsForgottenOnceItsRetentionPasses(t *testing.T) {
	server := miniredis.RunT(t)
	records := pageVisitRecordsOn(server, &manualClock{}, silentPageVisitRecords{})
	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	records.RecordPageVisit(context.Background(), url, pagefetch.PageVersion{})

	server.FastForward(retention)

	if visit, visited := records.LastPageVisitOf(context.Background(), url); visited {
		t.Fatalf("a page visit past its retention read back as %+v", visit)
	}
}
