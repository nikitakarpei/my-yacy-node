package companionpresence_test

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/companions/companionpresence"
)

const stepWait = 5 * time.Second

type changingConnections struct {
	devices map[companionprotocol.DeviceID]bool
	mu      sync.Mutex
}

func (c *changingConnections) ConnectedDevices() map[companionprotocol.DeviceID]bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return maps.Clone(c.devices)
}

func (c *changingConnections) connect(device companionprotocol.DeviceID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.devices[device] = true
}

func (c *changingConnections) disconnect(device companionprotocol.DeviceID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.devices, device)
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
	connections := &changingConnections{devices: map[companionprotocol.DeviceID]bool{}}
	clock := steppedClock{sleeps: make(chan chan struct{})}
	events := make(presenceEvents, 4)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- companionpresence.New(connections, clock, events).Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	clock.nextCheck(t)
	events.none(t)

	connections.connect("device-a")
	connections.connect("device-b")
	clock.nextCheck(t)
	clock.nextCheck(t)
	if event := <-events; event != "connected" {
		t.Fatalf("event = %q, want connected", event)
	}

	connections.disconnect("device-a")
	clock.nextCheck(t)
	clock.nextCheck(t)
	events.none(t)

	connections.disconnect("device-b")
	clock.nextCheck(t)
	clock.nextCheck(t)
	if event := <-events; event != "left" {
		t.Fatalf("event = %q, want left", event)
	}
}
