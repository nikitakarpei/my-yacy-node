package localhostruntunnel_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/internal/localhostruntunnel"
)

const forwardEvent = `{"event":"tcpip-forward","address":"e92b7b9260b6ff.lhr.life"}`

func TestTunnelReportsTheAssignedHostname(t *testing.T) {
	fakeSSH(t, forwardEvent, true)

	tunnel := opened(t, keylessConfiguration())

	if tunnel.Hostname() != "e92b7b9260b6ff.lhr.life" {
		t.Errorf("Hostname = %q, want e92b7b9260b6ff.lhr.life", tunnel.Hostname())
	}
}

func TestTunnelIgnoresEventsThatAreNotForwardEvents(t *testing.T) {
	fakeSSH(t, "starting\n{\"event\":\"other\"}\n"+forwardEvent, true)

	if hostname := opened(t, keylessConfiguration()).Hostname(); hostname == "" {
		t.Error("Hostname is empty, want the address of the forward event")
	}
}

func TestTunnelAsksForOneForwardWithoutAnIdentityFile(t *testing.T) {
	arguments := fakeSSH(t, forwardEvent, true)

	opened(t, keylessConfiguration())

	want := []string{
		"-T",
		"-o", "BatchMode=yes",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=60",
		"-o", "ServerAliveCountMax=3",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=/state/known_hosts",
		"-R", "80:127.0.0.1:34567",
		"nokey@localhost.run",
		"--",
		"--output", "json",
		"--no-inject-http-proxy-headers",
		"--proxy-protocol-header-version", "v2",
	}
	if given := recordedArguments(t, arguments); !slices.Equal(given, want) {
		t.Errorf("arguments = %v, want %v", given, want)
	}
}

func TestTunnelUsesTheIdentityFileAloneWhenItIsConfigured(t *testing.T) {
	arguments := fakeSSH(t, forwardEvent, true)

	configuration := keylessConfiguration()
	configuration.IdentityFilePath = "/state/identity"
	opened(t, configuration)

	given := recordedArguments(t, arguments)
	if !slices.Contains(given, "-i") || !slices.Contains(given, "/state/identity") {
		t.Errorf("arguments = %v, want the identity file", given)
	}
	if !slices.Contains(given, "IdentitiesOnly=yes") {
		t.Errorf("arguments = %v, want IdentitiesOnly=yes", given)
	}
	if slices.Contains(given, "nokey@localhost.run") {
		t.Errorf("arguments = %v, want the destination without the keyless user", given)
	}
}

func TestTunnelRejectsAnAddressThatIsNotAPublicHostname(t *testing.T) {
	for name, address := range map[string]string{
		"wildcard":      "*.lhr.life",
		"with a port":   "e92b7b9260b6ff.lhr.life:80",
		"with a path":   "e92b7b9260b6ff.lhr.life/admin",
		"with a scheme": "https://e92b7b9260b6ff.lhr.life",
		"one label":     "localhost",
		"empty":         "",
	} {
		t.Run(name, func(t *testing.T) {
			fakeSSH(t, `{"event":"tcpip-forward","address":"`+address+`"}`, true)

			_, err := localhostruntunnel.Open(t.Context(), keylessConfiguration())
			if err == nil {
				t.Fatalf("Open accepted the address %q", address)
			}
			if !errors.Is(err, localhostruntunnel.ErrBadEvent) {
				t.Errorf("Open error = %v, want ErrBadEvent", err)
			}
		})
	}
}

func TestTunnelReadsAForwardEventThatCarriesALongMessage(t *testing.T) {
	fakeSSH(t, `{"event":"tcpip-forward","address":"e92b7b9260b6ff.lhr.life","message":"`+
		strings.Repeat("q", 32*1024)+`"}`, true)

	tunnel := opened(t, keylessConfiguration())

	if tunnel.Hostname() != "e92b7b9260b6ff.lhr.life" {
		t.Errorf("Hostname = %q, want e92b7b9260b6ff.lhr.life", tunnel.Hostname())
	}
}

func TestTunnelFailsWhenTheProcessEndsBeforeTheForwardEvent(t *testing.T) {
	fakeSSH(t, "", false)

	_, err := localhostruntunnel.Open(t.Context(), keylessConfiguration())
	if err == nil {
		t.Fatal("Open accepted a process that ended without a forward event")
	}
	if !errors.Is(err, localhostruntunnel.ErrTunnelEnded) {
		t.Errorf("Open error = %v, want ErrTunnelEnded", err)
	}
}

func TestTunnelHoldsTheHostnameThroughARepeatedForwardEvent(t *testing.T) {
	fakeSSH(t, forwardEvent+"\n"+forwardEvent, true)

	tunnel := opened(t, keylessConfiguration())

	select {
	case err := <-tunnel.Failure():
		t.Fatalf("the tunnel failed on a repeated forward event: %v", err)
	case <-time.After(time.Second):
	}
	if tunnel.Hostname() != "e92b7b9260b6ff.lhr.life" {
		t.Errorf("Hostname = %q, want e92b7b9260b6ff.lhr.life", tunnel.Hostname())
	}
}

func TestTunnelReportsAReplacedHostnameAsAFailure(t *testing.T) {
	fakeSSH(t, forwardEvent+"\n"+
		`{"event":"tcpip-forward","address":"other-name.lhr.life"}`, true)

	tunnel := opened(t, keylessConfiguration())

	select {
	case err := <-tunnel.Failure():
		if !errors.Is(err, localhostruntunnel.ErrBadEvent) {
			t.Errorf("Failure = %v, want ErrBadEvent", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the tunnel accepted a second assigned hostname")
	}
}

func TestTunnelFailsWhenTheProcessEndsAfterTheForwardEvent(t *testing.T) {
	fakeSSH(t, forwardEvent, false)

	tunnel := opened(t, keylessConfiguration())

	select {
	case err := <-tunnel.Failure():
		if !errors.Is(err, localhostruntunnel.ErrTunnelEnded) {
			t.Errorf("Failure = %v, want ErrTunnelEnded", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the tunnel did not report the end of its process")
	}
}

func keylessConfiguration() localhostruntunnel.Configuration {
	return localhostruntunnel.Configuration{
		Host:           "localhost.run",
		KnownHostsPath: "/state/known_hosts",
		IngressPort:    34567,
	}
}

func opened(
	t *testing.T,
	configuration localhostruntunnel.Configuration,
) *localhostruntunnel.Tunnel {
	t.Helper()

	tunnel, err := localhostruntunnel.Open(t.Context(), configuration)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := tunnel.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	return tunnel
}

func fakeSSH(t *testing.T, events string, holds bool) string {
	t.Helper()

	directory := t.TempDir()
	arguments := filepath.Join(directory, "arguments")

	held := ""
	if holds {
		held = "exec sleep 60"
	}

	script := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + arguments + "\"\n" +
		"printf '%s\\n' \"$FAKE_SSH_EVENTS\"\n" +
		held + "\n"
	executable := filepath.Join(directory, "ssh")
	if err := os.WriteFile(executable, []byte(script), 0o600); err != nil {
		t.Fatalf("write the fake ssh: %v", err)
	}
	//nolint:gosec // the fake ssh must be executable to stand for the real one
	if err := os.Chmod(executable, 0o700); err != nil {
		t.Fatalf("make the fake ssh executable: %v", err)
	}

	t.Setenv("FAKE_SSH_EVENTS", events)
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))

	return arguments
}

func recordedArguments(t *testing.T, path string) []string {
	t.Helper()

	//nolint:gosec // the path is the temporary file of this test
	recorded, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the recorded arguments: %v", err)
	}

	return strings.Split(strings.TrimSuffix(string(recorded), "\n"), "\n")
}
