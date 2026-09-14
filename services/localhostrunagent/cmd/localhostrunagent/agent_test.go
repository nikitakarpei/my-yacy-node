package main_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	localhostrunagent "github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/cmd/localhostrunagent"
	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
)

func TestAgentLeasesTheAssignedHostnameToTheNode(t *testing.T) {
	configuration, _ := runningAgent(t)

	_, granted := leaseConsumer(t, configuration.LeaseSocketPath)

	if granted.ProcessEnvironment["YACY_ADVERTISE_HOST"] != "tidy-mouse-42.localhost.run" {
		t.Errorf("YACY_ADVERTISE_HOST = %q, want tidy-mouse-42.localhost.run",
			granted.ProcessEnvironment["YACY_ADVERTISE_HOST"])
	}
	if granted.ProcessEnvironment["YACY_ADVERTISE_PORT"] != "80" {
		t.Errorf("YACY_ADVERTISE_PORT = %q, want 80",
			granted.ProcessEnvironment["YACY_ADVERTISE_PORT"])
	}
}

func TestAgentStopsWhenTheNodeEndsTheLease(t *testing.T) {
	configuration, failure := runningAgent(t)

	consumer, _ := leaseConsumer(t, configuration.LeaseSocketPath)
	if err := consumer.Close(); err != nil {
		t.Fatalf("close the lease consumer: %v", err)
	}

	if err := <-failure; err == nil {
		t.Error("RunAgent returned no error after the node ended the lease")
	}
}

func TestAgentStopsWhenItIsSignalled(t *testing.T) {
	configuration, failure, stop := startedAgent(t)
	leaseConsumer(t, configuration.LeaseSocketPath)

	stop()

	select {
	case err := <-failure:
		if err != nil {
			t.Errorf("RunAgent = %v, want no error after a signal", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunAgent did not return after a signal")
	}
}

func runningAgent(t *testing.T) (localhostrunagent.AgentConfiguration, chan error) {
	t.Helper()

	configuration, failure, _ := startedAgent(t)

	return configuration, failure
}

func startedAgent(
	t *testing.T,
) (localhostrunagent.AgentConfiguration, chan error, context.CancelFunc) {
	t.Helper()

	fakeSSH(t)

	configuration := localhostrunagent.AgentConfiguration{
		NodeOrigin:       nodeOrigin(t),
		LeaseSocketPath:  leaseSocketPath(t),
		LocalhostRunHost: "localhost.run",
		KnownHostsPath:   filepath.Join(t.TempDir(), "known_hosts"),
		AgentAddress:     netip.MustParseAddr("127.0.0.1"),
	}

	ctx, stop := context.WithCancel(t.Context())
	failure := make(chan error, 1)

	go func() { failure <- localhostrunagent.RunAgent(ctx, configuration) }()

	t.Cleanup(stop)

	return configuration, failure, stop
}

func nodeOrigin(t *testing.T) *url.URL {
	t.Helper()

	server := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse the node origin: %v", err)
	}

	return parsed
}

func leaseSocketPath(t *testing.T) string {
	t.Helper()

	directory, err := os.MkdirTemp("", "lease")
	if err != nil {
		t.Fatalf("make the lease socket directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })

	return filepath.Join(directory, "process-environment-lease.sock")
}

func fakeSSH(t *testing.T) {
	t.Helper()

	directory := t.TempDir()
	executable := filepath.Join(directory, "ssh")

	script := "#!/bin/sh\n" +
		`printf '%s\n' '{"event":"tcpip-forward","address":"tidy-mouse-42.localhost.run"}'` +
		"\nexec sleep 60\n"
	if err := os.WriteFile(executable, []byte(script), 0o600); err != nil {
		t.Fatalf("write the fake ssh: %v", err)
	}
	//nolint:gosec // the fake ssh must be executable to stand for the real one
	if err := os.Chmod(executable, 0o700); err != nil {
		t.Fatalf("make the fake ssh executable: %v", err)
	}

	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func leaseConsumer(
	t *testing.T,
	socketPath string,
) (net.Conn, processenvironmentlease.Grant) {
	t.Helper()

	dialer := net.Dialer{Timeout: 10 * time.Second}
	deadline := time.Now().Add(30 * time.Second)

	for {
		consumer, granted, err := grantedLease(t.Context(), dialer, socketPath)
		if err == nil {
			t.Cleanup(func() { _ = consumer.Close() })

			return consumer, granted
		}
		if time.Now().After(deadline) {
			t.Fatalf("take the process environment lease: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func grantedLease(
	ctx context.Context,
	dialer net.Dialer,
	socketPath string,
) (net.Conn, processenvironmentlease.Grant, error) {
	consumer, err := dialer.DialContext(ctx, "unix", socketPath)
	if err != nil {
		return nil, processenvironmentlease.Grant{}, fmt.Errorf("dial the lease socket: %w", err)
	}

	granted, err := grantRead(consumer)
	if err != nil {
		_ = consumer.Close()

		return nil, processenvironmentlease.Grant{}, err
	}

	return consumer, granted, nil
}

func grantRead(consumer net.Conn) (processenvironmentlease.Grant, error) {
	frame := make([]byte, processenvironmentlease.MaximumFrameByte)

	if err := consumer.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return processenvironmentlease.Grant{}, fmt.Errorf("set the read deadline: %w", err)
	}

	read, err := consumer.Read(frame)
	if err != nil {
		return processenvironmentlease.Grant{}, fmt.Errorf("read the grant frame: %w", err)
	}

	granted, err := processenvironmentlease.GrantFrom(frame[:read])
	if err != nil {
		return processenvironmentlease.Grant{}, fmt.Errorf("read the grant frame: %w", err)
	}

	return granted, nil
}
