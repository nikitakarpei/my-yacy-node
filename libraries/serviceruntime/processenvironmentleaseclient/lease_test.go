package processenvironmentleaseclient_test

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/processenvironmentleaseclient"
)

const acquisitionBudget = 500 * time.Millisecond

func TestLeaseCarriesTheGrant(t *testing.T) {
	socketPath, producer := producing(t, `{"protocol_version":1,"process_environment":`+
		`{"YACY_ADVERTISE_HOST":"name.localhost.run"}}`+"\n")
	defer producer.close()

	lease := acquired(t, socketPath)
	defer closeLease(t, lease)

	if host := lease.Grant().ProcessEnvironment["YACY_ADVERTISE_HOST"]; host != "name.localhost.run" {
		t.Errorf("YACY_ADVERTISE_HOST = %q, want %q", host, "name.localhost.run")
	}
}

func TestLeaseIsRevokedWhenTheProducerCloses(t *testing.T) {
	socketPath, producer := producing(t, `{"protocol_version":1,"process_environment":{}}`+"\n")

	lease := acquired(t, socketPath)
	defer closeLease(t, lease)

	producer.close()

	if err := revocation(t, lease); !errors.Is(err, processenvironmentleaseclient.ErrRevoked) {
		t.Errorf("Revocation = %v, want ErrRevoked", err)
	}
}

func TestLeaseIsRevokedWhenTheProducerWritesAfterTheGrant(t *testing.T) {
	socketPath, producer := producing(t,
		`{"protocol_version":1,"process_environment":{}}`+"\n"+
			`{"protocol_version":1,"process_environment":{}}`+"\n",
	)
	defer producer.close()

	lease := acquired(t, socketPath)
	defer closeLease(t, lease)

	err := revocation(t, lease)
	if !errors.Is(err, processenvironmentleaseclient.ErrRevoked) {
		t.Errorf("Revocation = %v, want ErrRevoked", err)
	}
	if !errors.Is(err, processenvironmentlease.ErrBadGrant) {
		t.Errorf("Revocation = %v, want ErrBadGrant", err)
	}
}

func TestLeaseKeepsTryingWhileTheProducerIsAbsent(t *testing.T) {
	socketPath := socketPathIn(t)

	ctx, cancel := context.WithTimeout(t.Context(), acquisitionBudget)
	defer cancel()

	begun := time.Now()
	if _, err := processenvironmentleaseclient.Acquire(ctx, socketPath); err == nil {
		t.Fatal("Acquire returned a lease with no producer listening")
	}

	if waited := time.Since(begun); waited < acquisitionBudget {
		t.Errorf(
			"Acquire gave up after %v, want it to keep trying for %v",
			waited,
			acquisitionBudget,
		)
	}
}

func TestLeaseIsAcquiredAfterAProducerSendsABadGrant(t *testing.T) {
	socketPath, producer := producing(
		t,
		"not a grant\n",
		`{"protocol_version":1,"process_environment":{"YACY_ADVERTISE_HOST":"name.localhost.run"}}`+"\n",
	)
	defer producer.close()

	lease := acquired(t, socketPath)
	defer closeLease(t, lease)

	if host := lease.Grant().ProcessEnvironment["YACY_ADVERTISE_HOST"]; host != "name.localhost.run" {
		t.Errorf("YACY_ADVERTISE_HOST = %q, want %q", host, "name.localhost.run")
	}
}

func TestLeaseRetriesAProducerThatSendsNoGrant(t *testing.T) {
	socketPath := socketPathIn(t)
	silent := listening(t, socketPath, "")
	defer silent.close()

	ctx, cancel := context.WithTimeout(t.Context(), acquisitionBudget)
	defer cancel()

	begun := time.Now()
	if _, err := processenvironmentleaseclient.Acquire(ctx, socketPath); err == nil {
		t.Fatal("Acquire returned a lease without a grant")
	}

	if waited := time.Since(begun); waited > 2*acquisitionBudget {
		t.Errorf("Acquire read the missing grant for %v, want it to end with the context", waited)
	}
}

type producer struct {
	listener net.Listener
	accepted []net.Conn
	stopped  chan struct{}
}

func producing(t *testing.T, frames ...string) (string, *producer) {
	t.Helper()

	socketPath := socketPathIn(t)

	return socketPath, listening(t, socketPath, frames...)
}

func socketPathIn(t *testing.T) string {
	t.Helper()

	directory, err := os.MkdirTemp("/tmp", "lease")
	if err != nil {
		t.Fatalf("make the socket directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("remove the socket directory: %v", err)
		}
	})

	return filepath.Join(directory, "lease.sock")
}

func listening(t *testing.T, socketPath string, frames ...string) *producer {
	t.Helper()

	configuration := net.ListenConfig{}

	listener, err := configuration.Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	started := &producer{listener: listener, stopped: make(chan struct{})}
	go started.serve(frames)

	return started
}

func (started *producer) serve(frames []string) {
	defer close(started.stopped)

	for {
		connection, err := started.listener.Accept()
		if err != nil {
			return
		}
		started.accepted = append(started.accepted, connection)

		frame := frames[min(len(started.accepted)-1, len(frames)-1)]
		if frame != "" {
			_, _ = connection.Write([]byte(frame))
		}
	}
}

func (started *producer) close() {
	_ = started.listener.Close()
	<-started.stopped

	for _, connection := range started.accepted {
		_ = connection.Close()
	}
}

func acquired(t *testing.T, socketPath string) *processenvironmentleaseclient.Lease {
	t.Helper()

	lease, err := processenvironmentleaseclient.Acquire(t.Context(), socketPath)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	return lease
}

func closeLease(t *testing.T, lease *processenvironmentleaseclient.Lease) {
	t.Helper()

	if err := lease.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func revocation(t *testing.T, lease *processenvironmentleaseclient.Lease) error {
	t.Helper()

	select {
	case err := <-lease.Revocation():
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("the lease was not revoked")

		return nil
	}
}
