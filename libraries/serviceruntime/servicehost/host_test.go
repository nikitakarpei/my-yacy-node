package servicehost_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehost"
)

const runWait = 10 * time.Second

var errServiceFailed = errors.New("the service failed")

func TestRunKeepsTheServiceRunningUntilStop(t *testing.T) {
	serviceStarted := make(chan struct{})
	host := servicehost.HostFor(
		"127.0.0.1:0",
		func(ctx context.Context, _ *prometheus.Registry) error {
			close(serviceStarted)
			<-ctx.Done()
			return nil
		},
	)
	runDone := make(chan error, 1)
	go func() { runDone <- host.Run() }()

	<-serviceStarted
	host.Stop()

	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("Run: %v", err)
		}
	case <-time.After(runWait):
		t.Fatal("Run did not return after Stop")
	}
}

func TestRunReportsWhatTheServiceFailedWith(t *testing.T) {
	host := servicehost.HostFor("127.0.0.1:0", func(context.Context, *prometheus.Registry) error {
		return errServiceFailed
	})

	if err := host.Run(); !errors.Is(err, errServiceFailed) {
		t.Errorf("Run = %v, want %v", err, errServiceFailed)
	}
}

func TestRunReportsAnOpsAddressItCannotListenOn(t *testing.T) {
	host := servicehost.HostFor(
		"not an address",
		func(context.Context, *prometheus.Registry) error {
			t.Error("the service ran without its ops endpoint")
			return nil
		},
	)

	if err := host.Run(); err == nil {
		t.Fatal("want an error for an ops address the host cannot listen on")
	}
}

func TestStopBeforeRunEndsTheRunAtOnce(t *testing.T) {
	host := servicehost.HostFor(
		"127.0.0.1:0",
		func(ctx context.Context, _ *prometheus.Registry) error {
			<-ctx.Done()
			return nil
		},
	)
	host.Stop()
	host.Stop()

	if err := host.Run(); err != nil {
		t.Errorf("Run: %v", err)
	}
}
