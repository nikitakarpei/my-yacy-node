package processenvironmentleaseserver_test

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/processenvironmentleaseserver"
)

func TestServerGrantsTheProcessEnvironmentToTheFirstConsumerAfterTheGrant(t *testing.T) {
	socketPath := socketPathIn(t)
	server := listening(t, socketPath)

	server.Grant(processenvironmentlease.Grant{
		ProcessEnvironment: map[string]string{"YACY_ADVERTISE_HOST": "name.localhost.run"},
	})

	granted := grantRead(t, dialed(t, socketPath))
	if granted.ProcessEnvironment["YACY_ADVERTISE_HOST"] != "name.localhost.run" {
		t.Errorf("ProcessEnvironment = %v, want the granted host", granted.ProcessEnvironment)
	}
}

func TestServerClosesAConsumerThatConnectsBeforeTheGrant(t *testing.T) {
	socketPath := socketPathIn(t)
	listening(t, socketPath)

	consumer := dialed(t, socketPath)
	if err := readDeadlined(t, consumer); !errors.Is(err, io.EOF) {
		t.Errorf("read before the grant = %v, want EOF", err)
	}
}

func TestServerStopsListeningAfterTheGrant(t *testing.T) {
	socketPath := socketPathIn(t)
	server := listening(t, socketPath)

	server.Grant(processenvironmentlease.Grant{})
	grantRead(t, dialed(t, socketPath))

	dialer := net.Dialer{Timeout: time.Second}

	if _, err := dialer.DialContext(t.Context(), "unix", socketPath); err == nil {
		t.Error("the lease socket accepted a second consumer")
	}
}

func TestServerReportsTheEndOfTheConsumer(t *testing.T) {
	socketPath := socketPathIn(t)
	server := listening(t, socketPath)

	server.Grant(processenvironmentlease.Grant{})
	consumer := dialed(t, socketPath)
	grantRead(t, consumer)

	if err := consumer.Close(); err != nil {
		t.Fatalf("close the consumer: %v", err)
	}

	select {
	case err := <-server.Failure():
		if !errors.Is(err, processenvironmentleaseserver.ErrConsumerEnded) {
			t.Errorf("Failure = %v, want ErrConsumerEnded", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the server did not report the end of the consumer")
	}
}

func TestServerRefusesASocketPathWithAnActiveListener(t *testing.T) {
	socketPath := socketPathIn(t)
	listening(t, socketPath)

	_, err := processenvironmentleaseserver.Listen(t.Context(), socketPath)
	if err == nil {
		t.Fatal("Listen accepted a socket path with an active listener")
	}
	if !errors.Is(err, processenvironmentleaseserver.ErrSocketInUse) {
		t.Errorf("Listen error = %v, want ErrSocketInUse", err)
	}
}

func TestServerListensOnAStaleSocketPath(t *testing.T) {
	socketPath := socketPathIn(t)
	if err := os.WriteFile(socketPath, nil, 0o600); err != nil {
		t.Fatalf("write the stale socket path: %v", err)
	}

	server := listening(t, socketPath)
	server.Grant(processenvironmentlease.Grant{})
	grantRead(t, dialed(t, socketPath))
}

func socketPathIn(t *testing.T) string {
	t.Helper()

	directory, err := os.MkdirTemp("", "lease")
	if err != nil {
		t.Fatalf("make the lease socket directory: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(directory) })

	return filepath.Join(directory, "process-environment-lease.sock")
}

func listening(t *testing.T, socketPath string) *processenvironmentleaseserver.Server {
	t.Helper()

	server, err := processenvironmentleaseserver.Listen(t.Context(), socketPath)
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	return server
}

func dialed(t *testing.T, socketPath string) net.Conn {
	t.Helper()

	dialer := net.Dialer{Timeout: 10 * time.Second}

	consumer, err := dialer.DialContext(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("dial the lease socket: %v", err)
	}
	t.Cleanup(func() { _ = consumer.Close() })

	return consumer
}

func grantRead(t *testing.T, consumer net.Conn) processenvironmentlease.Grant {
	t.Helper()

	frame := make([]byte, processenvironmentlease.MaximumFrameByte)

	if err := consumer.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set the read deadline: %v", err)
	}

	read, err := consumer.Read(frame)
	if err != nil {
		t.Fatalf("read the grant frame: %v", err)
	}

	granted, err := processenvironmentlease.GrantFrom(frame[:read])
	if err != nil {
		t.Fatalf("GrantFrom: %v", err)
	}

	return granted
}

func readDeadlined(t *testing.T, consumer net.Conn) error {
	t.Helper()

	if err := consumer.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set the read deadline: %v", err)
	}

	_, err := consumer.Read(make([]byte, 1))

	return fmt.Errorf("read from the lease socket: %w", err)
}
