package servergroup_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func listeningServer(t *testing.T) servergroup.NamedServer {
	t.Helper()
	listener, err := servergroup.Listen(context.Background(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return servergroup.NamedServer{
		Name:     "test",
		Listener: listener,
		Server: &http.Server{
			Handler:           http.NewServeMux(),
			ReadHeaderTimeout: time.Second,
		},
	}
}

func TestContextCancelShutsDown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- servergroup.Run(ctx, time.Second, []servergroup.NamedServer{listeningServer(t)})
	}()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestFirstWorkerReturnShutsDownServers(t *testing.T) {
	server := listeningServer(t)
	worker := func(context.Context) error { return nil }

	done := make(chan error, 1)
	go func() {
		done <- servergroup.Run(context.Background(), time.Second,
			[]servergroup.NamedServer{server}, worker)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker return did not shut down the group")
	}
}

func TestWorkerErrorPropagates(t *testing.T) {
	wantErr := errors.New("worker failed")
	blocking := func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}
	failing := func(context.Context) error { return wantErr }

	err := servergroup.Run(context.Background(), time.Second,
		[]servergroup.NamedServer{listeningServer(t)}, blocking, failing)
	if !errors.Is(err, wantErr) {
		t.Errorf("Run = %v, want %v", err, wantErr)
	}
}

func TestWorkerReportingTheCancellationHasStopped(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := servergroup.Run(ctx, time.Second, nil, func(ctx context.Context) error {
		<-ctx.Done()
		return fmt.Errorf("open the store: %w", ctx.Err())
	})
	if err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
}

func TestWorkerCancelledWithoutTheCallerFails(t *testing.T) {
	err := servergroup.Run(context.Background(), time.Second, nil, func(context.Context) error {
		return fmt.Errorf("open the store: %w", context.Canceled)
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Run = %v, want the worker's cancellation", err)
	}
}

func TestListenReportsAnAddressInUse(t *testing.T) {
	holder, err := servergroup.Listen(context.Background(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = holder.Close() }()

	if _, err := servergroup.Listen(context.Background(), holder.Addr().String()); err == nil {
		t.Error("Listen = nil, want the address in use")
	}
}

func TestServeErrorPropagates(t *testing.T) {
	server := listeningServer(t)
	if err := server.Listener.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if err := servergroup.Run(context.Background(), time.Second,
		[]servergroup.NamedServer{server}); err == nil {
		t.Error("Run = nil, want the serve error")
	}
}

type runValueKey struct{}

func TestRequestsCarryTheRunValuesButOutliveItsCancellation(t *testing.T) {
	server := listeningServer(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	type requestContextState struct {
		value any
		err   error
	}
	seen := make(chan requestContextState, 1)
	server.Server.Handler = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		seen <- requestContextState{value: r.Context().Value(runValueKey{}), err: r.Context().Err()}
	})
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), runValueKey{}, "run"))
	done := make(chan error, 1)
	go func() { done <- servergroup.Run(ctx, 5*time.Second, []servergroup.NamedServer{server}) }()
	go func() {
		request, err := http.NewRequestWithContext(
			context.Background(), http.MethodGet, "http://"+server.Listener.Addr().String(), nil,
		)
		if err != nil {
			return
		}
		if response, err := http.DefaultClient.Do(request); err == nil {
			_ = response.Body.Close()
		}
	}()

	<-entered
	cancel()
	close(release)
	state := <-seen
	if state.value != "run" {
		t.Error("request context lacks the run value")
	}
	if state.err != nil {
		t.Errorf("request context ended with the run: %v", state.err)
	}
	if err := <-done; err != nil {
		t.Errorf("Run: %v", err)
	}
}
