package jetstream_test

import (
	"context"
	"errors"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	jetstreamrecord "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
	visitedpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/visitedpages/jetstream"
)

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func newBucket(t *testing.T) natsjetstream.KeyValue {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if err := visitedpagesjetstream.Ensure(
		context.Background(),
		js,
		jetstreamrecord.BucketSpec{MaxBytes: 1 << 20, Retention: time.Hour},
	); err != nil {
		t.Fatal(err)
	}
	bucket, err := js.KeyValue(context.Background(), visitedpagesjetstream.BucketName)
	if err != nil {
		t.Fatal(err)
	}
	return bucket
}

func TestNoPageVisitOfAPageNeverVisited(t *testing.T) {
	pages := visitedpagesjetstream.New(newBucket(t), &manualClock{}, silentPageVisitRecords{})

	visit, visited, err := pages.LastPageVisitOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)
	if err != nil {
		t.Fatalf("last page visit: %v", err)
	}
	if visited {
		t.Fatalf("want no page visit, got %+v", visit)
	}
}

func TestARecordedPageVisitKeepsItsTimeAndPageVersion(t *testing.T) {
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{now: now}
	pages := visitedpagesjetstream.New(newBucket(t), clock, silentPageVisitRecords{})

	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	version := pagefetch.PageVersion{EntityTag: `"abc"`, ModifiedAt: now.Add(-time.Minute)}
	pages.RecordPageVisit(context.Background(), url, version)

	clock.now = now.Add(30 * time.Minute)
	visit, visited, err := pages.LastPageVisitOf(context.Background(), url)
	if err != nil {
		t.Fatalf("last page visit: %v", err)
	}
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

type recordedFailures struct{ causes []error }

func (f *recordedFailures) PageVisitNotRecorded(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	f.causes = append(f.causes, cause)
}

type failingBucket struct {
	natsjetstream.KeyValue
	getErr error
	putErr error
}

func (f failingBucket) Get(context.Context, string) (natsjetstream.KeyValueEntry, error) {
	return nil, f.getErr
}

func (f failingBucket) Put(context.Context, string, []byte) (uint64, error) {
	return 0, f.putErr
}

func TestALookupTheBucketRefusesFailsFast(t *testing.T) {
	boom := errors.New("boom")
	pages := visitedpagesjetstream.New(
		failingBucket{getErr: boom},
		&manualClock{},
		silentPageVisitRecords{},
	)

	if _, _, err := pages.LastPageVisitOf(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want the refusal of the bucket", err)
	}
}

func TestAPageVisitTheBucketRefusesIsReported(t *testing.T) {
	boom := errors.New("bucket down")
	failures := &recordedFailures{}
	pages := visitedpagesjetstream.New(
		failingBucket{putErr: boom}, &manualClock{}, failures,
	)

	pages.RecordPageVisit(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
		pagefetch.PageVersion{},
	)

	if len(failures.causes) != 1 || !errors.Is(failures.causes[0], boom) {
		t.Fatalf("reported %v, want the refusal of the bucket", failures.causes)
	}
}
