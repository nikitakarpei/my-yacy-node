package dueaftergrace_test

import (
	"context"
	"errors"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/recrawldecisions/dueaftergrace"
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

func TestRevisitDueWithNoValidatorsWhenNeverVisited(t *testing.T) {
	rule := dueaftergrace.New(newBucket(t), &manualClock{}, time.Hour)

	revisit, err := rule.Revisit(context.Background(), "http://example.com/a")
	if err != nil {
		t.Fatalf("revisit: %v", err)
	}
	if !revisit.Due || revisit.EntityTag != "" || !revisit.ModifiedAt.IsZero() {
		t.Fatalf("want due with no validators, got %+v", revisit)
	}
}

func TestRevisitNotDueWithinGraceButReturnsValidators(t *testing.T) {
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{now: now}
	rule := dueaftergrace.New(newBucket(t), clock, time.Hour)

	const url = "http://example.com/a"
	validators := pagevisit.Revisit{EntityTag: `"abc"`, ModifiedAt: now.Add(-time.Minute)}
	if err := rule.Visited(context.Background(), url, validators); err != nil {
		t.Fatalf("visited: %v", err)
	}

	clock.now = now.Add(30 * time.Minute)
	revisit, err := rule.Revisit(context.Background(), url)
	if err != nil {
		t.Fatalf("revisit: %v", err)
	}
	if revisit.Due {
		t.Fatal("want not due within grace window")
	}
	if revisit.EntityTag != validators.EntityTag ||
		!revisit.ModifiedAt.Equal(validators.ModifiedAt) {
		t.Fatalf("validators = %+v, want %+v", revisit, validators)
	}
}

func TestRevisitDueOutsideGraceWindow(t *testing.T) {
	now := time.Date(2026, time.July, 25, 12, 0, 0, 0, time.UTC)
	clock := &manualClock{now: now}
	rule := dueaftergrace.New(newBucket(t), clock, time.Hour)

	const url = "http://example.com/a"
	if err := rule.Visited(
		context.Background(),
		url,
		pagevisit.Revisit{EntityTag: `"abc"`},
	); err != nil {
		t.Fatalf("visited: %v", err)
	}

	clock.now = now.Add(2 * time.Hour)
	revisit, err := rule.Revisit(context.Background(), url)
	if err != nil {
		t.Fatalf("revisit: %v", err)
	}
	if !revisit.Due {
		t.Fatal("want due outside grace window")
	}
	if revisit.EntityTag != `"abc"` {
		t.Fatalf("entity tag = %q, want validators still returned", revisit.EntityTag)
	}
}

type failingBucket struct {
	natsjetstream.KeyValue
	getErr error
}

func (f failingBucket) Get(context.Context, string) (natsjetstream.KeyValueEntry, error) {
	return nil, f.getErr
}

func TestRevisitPropagatesOtherErrors(t *testing.T) {
	boom := errors.New("boom")
	rule := dueaftergrace.New(failingBucket{getErr: boom}, &manualClock{}, time.Hour)

	if _, err := rule.Revisit(context.Background(), "http://example.com/a"); err == nil {
		t.Fatal("want error propagated")
	}
}
