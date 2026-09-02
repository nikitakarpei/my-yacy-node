package processentrypoint_test

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/processentrypoint"
)

func TestEntrypointReportsTheCommandExitStatus(t *testing.T) {
	status, err := processentrypoint.Run(
		t.Context(), []string{"/bin/sh", "-c", "exit 3"}, unset, nil,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if status != 3 {
		t.Errorf("Run = %d, want 3", status)
	}
}

func TestEntrypointRejectsAnEmptyCommand(t *testing.T) {
	_, err := processentrypoint.Run(t.Context(), nil, unset, nil)
	if !errors.Is(err, processentrypoint.ErrNoCommand) {
		t.Errorf("Run error = %v, want ErrNoCommand", err)
	}
}

func TestEntrypointRunsTheCommandWithTheLeasedEnvironment(t *testing.T) {
	socketPath := granting(t, map[string]string{"LEASED_NAME": "leased.example"})

	status, err := processentrypoint.Run(
		t.Context(),
		[]string{"/bin/sh", "-c", `test "$LEASED_NAME" = leased.example && test "$KEPT" = kept`},
		leaseSocketAt(socketPath),
		[]string{"KEPT=kept", "LEASED_NAME=inherited.example"},
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if status != 0 {
		t.Errorf("Run = %d, want 0", status)
	}
}

func TestEntrypointStopsTheCommandWhenTheLeaseIsRevoked(t *testing.T) {
	socketPath := granting(t, map[string]string{})

	status, err := processentrypoint.Run(
		t.Context(),
		[]string{"sleep", "60"},
		leaseSocketAt(socketPath),
		nil,
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if status == 0 {
		t.Errorf("Run = %d, want the status of a stopped command", status)
	}
}

func unset(string) string { return "" }

func leaseSocketAt(socketPath string) func(string) string {
	return func(name string) string {
		if name == processentrypoint.EnvProcessEnvironmentLeaseSocket {
			return socketPath
		}

		return ""
	}
}

func granting(t *testing.T, processEnvironment map[string]string) string {
	t.Helper()

	frame, err := processenvironmentlease.FrameOf(
		processenvironmentlease.Grant{ProcessEnvironment: processEnvironment},
	)
	if err != nil {
		t.Fatalf("FrameOf: %v", err)
	}

	socketPath := socketPathIn(t)

	configuration := net.ListenConfig{}

	listener, err := configuration.Listen(t.Context(), "unix", socketPath)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go grant(listener, frame)

	return socketPath
}

func grant(listener net.Listener, frame []byte) {
	connection, err := listener.Accept()
	if err != nil {
		return
	}
	defer func() { _ = connection.Close() }()

	if _, err := connection.Write(frame); err != nil {
		return
	}

	time.Sleep(500 * time.Millisecond)
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
