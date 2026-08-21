package dueaftergrace_test

import (
	"context"
	"errors"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawlrules/dueaftergrace"
)

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func (c *manualClock) Sleep(context.Context, time.Duration) error { return nil }

func newBucket(t *testing.T) natsjetstream.KeyValue {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if err := dueaftergrace.Ensure(
		context.Background(), js, dueaftergrace.BucketSpec{MaxBytes: 1 << 20, Retention: time.Hour},
	); err != nil {
		t.Fatal(err)
	}
	bucket, err := js.KeyValue(context.Background(), dueaftergrace.BucketName)
	if err != nil {
		t.Fatal(err)
	}
	return bucket
}

func TestDecisionDueWithNoVersionWhenNeverVisited(t *testing.T) {
	rule := dueaftergrace.New(newBucket(t), &manualClock{}, time.Hour)

	decision, err := rule.DecisionFor(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}
	if !decision.Due || decision.Version != (pagefetch.PageVersion{}) {
		t.Fatalf("want due with no page version, got %+v", decision)
	}
}

func TestDecisionNotDueWithinGraceButReturnsVersion(t *testing.T) {
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{now: now}
	rule := dueaftergrace.New(newBucket(t), clock, time.Hour)

	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	version := pagefetch.PageVersion{EntityTag: `"abc"`, ModifiedAt: now.Add(-time.Minute)}
	if err := rule.RecordVisit(context.Background(), url, version); err != nil {
		t.Fatalf("record visit: %v", err)
	}

	clock.now = now.Add(30 * time.Minute)
	decision, err := rule.DecisionFor(context.Background(), url)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}
	if decision.Due {
		t.Fatal("want not due within grace window")
	}
	if decision.Version.EntityTag != version.EntityTag ||
		!decision.Version.ModifiedAt.Equal(version.ModifiedAt) {
		t.Fatalf("page version = %+v, want %+v", decision.Version, version)
	}
}

func TestDecisionDueOutsideGraceWindow(t *testing.T) {
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{now: now}
	rule := dueaftergrace.New(newBucket(t), clock, time.Hour)

	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	if err := rule.RecordVisit(
		context.Background(),
		url,
		pagefetch.PageVersion{EntityTag: `"abc"`},
	); err != nil {
		t.Fatalf("record visit: %v", err)
	}

	clock.now = now.Add(2 * time.Hour)
	decision, err := rule.DecisionFor(context.Background(), url)
	if err != nil {
		t.Fatalf("decision: %v", err)
	}
	if !decision.Due {
		t.Fatal("want due outside grace window")
	}
	if decision.Version.EntityTag != `"abc"` {
		t.Fatalf("entity tag = %q, want the stored version still returned",
			decision.Version.EntityTag)
	}
}

type failingBucket struct {
	natsjetstream.KeyValue
	getErr error
}

func (f failingBucket) Get(context.Context, string) (natsjetstream.KeyValueEntry, error) {
	return nil, f.getErr
}

func TestDecisionPropagatesOtherErrors(t *testing.T) {
	boom := errors.New("boom")
	rule := dueaftergrace.New(failingBucket{getErr: boom}, &manualClock{}, time.Hour)

	if _, err := rule.DecisionFor(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, "http://example.com/a"),
	); err == nil {
		t.Fatal("want error propagated")
	}
}
