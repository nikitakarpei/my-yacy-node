package companionpresence_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionaccess"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionpresence"
)

const stepWait = 5 * time.Second

type changingBroker struct {
	users map[string]bool
	mu    sync.Mutex
}

func (b *changingBroker) ConnectedUsers() map[string]bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	users := make(map[string]bool, len(b.users))
	for user := range b.users {
		users[user] = true
	}
	return users
}

func (b *changingBroker) connect(user string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.users[user] = true
}

func (b *changingBroker) disconnect(user string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.users, user)
}

type steppedClock struct {
	sleeps chan chan struct{}
}

func (c steppedClock) Sleep(ctx context.Context, _ time.Duration) error {
	woken := make(chan struct{})
	select {
	case c.sleeps <- woken:
	case <-ctx.Done():
		return fmt.Errorf("sleep: %w", ctx.Err())
	}
	select {
	case <-woken:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("sleep: %w", ctx.Err())
	}
}

func (c steppedClock) nextCheck(t *testing.T) {
	t.Helper()
	select {
	case woken := <-c.sleeps:
		close(woken)
	case <-time.After(stepWait):
		t.Fatal("the presence did not wait for its next check")
	}
}

type presenceEvents chan string

func (e presenceEvents) CompanionsConnected(context.Context) { e <- "connected" }

func (e presenceEvents) CompanionsLeft(context.Context) { e <- "left" }

func (e presenceEvents) none(t *testing.T) {
	t.Helper()
	select {
	case event := <-e:
		t.Fatalf("event = %q, want none", event)
	default:
	}
}

func TestPresenceTellsWhenTheFirstCompanionConnectsAndTheLastOneLeaves(t *testing.T) {
	broker := &changingBroker{users: map[string]bool{"peer-service": true}}
	clock := steppedClock{sleeps: make(chan chan struct{})}
	events := make(presenceEvents, 4)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- companionpresence.New(broker, clock, events).Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	clock.nextCheck(t)
	events.none(t)

	broker.connect(companionaccess.UserOf("device-a"))
	broker.connect(companionaccess.UserOf("device-b"))
	clock.nextCheck(t)
	clock.nextCheck(t)
	if event := <-events; event != "connected" {
		t.Fatalf("event = %q, want connected", event)
	}

	broker.disconnect(companionaccess.UserOf("device-a"))
	clock.nextCheck(t)
	clock.nextCheck(t)
	events.none(t)

	broker.disconnect(companionaccess.UserOf("device-b"))
	clock.nextCheck(t)
	clock.nextCheck(t)
	if event := <-events; event != "left" {
		t.Fatalf("event = %q, want left", event)
	}
}
