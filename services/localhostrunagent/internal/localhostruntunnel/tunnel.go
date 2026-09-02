// Package localhostruntunnel owns the OpenSSH process that holds one
// localhost.run reverse tunnel, the bounded JSON event lines that process
// writes, and the public hostname localhost.run assigns to the tunnel. It
// reports the end of the tunnel as a failure, because there is no inner retry.
package localhostruntunnel

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

const (
	executableName       = "ssh"
	keylessUser          = "nokey"
	acquisitionTimeout   = 60 * time.Second
	terminationGrace     = 8 * time.Second
	maximumEventLineByte = 64 * 1024
	PublicPort           = 80
	loopbackAddress      = "127.0.0.1"
)

var (
	ErrNoAssignedHostname = errors.New("no hostname assigned to the tunnel")
	ErrTunnelEnded        = errors.New("the tunnel ended")
)

type Configuration struct {
	Host             string
	IdentityFilePath string
	KnownHostsPath   string
	IngressPort      int
}

type Tunnel struct {
	command  *exec.Cmd
	stop     context.CancelFunc
	assigned chan string
	failure  chan error
	hostname string
}

func Open(ctx context.Context, configuration Configuration) (*Tunnel, error) {
	running, stop := context.WithCancel(context.WithoutCancel(ctx))

	tunnel := &Tunnel{
		command:  terminableCommand(running, configuration),
		stop:     stop,
		assigned: make(chan string, 1),
		failure:  make(chan error, 1),
	}

	events, err := tunnel.command.StdoutPipe()
	if err != nil {
		stop()

		return nil, fmt.Errorf("read the tunnel events: %w", err)
	}

	if err := tunnel.command.Start(); err != nil {
		stop()

		return nil, fmt.Errorf("start the tunnel: %w", err)
	}

	go tunnel.read(context.WithoutCancel(ctx), eventLines(events))

	hostname, err := tunnel.acquiredHostname(ctx)
	if err != nil {
		stop()

		return nil, err
	}
	tunnel.hostname = hostname

	slog.DebugContext(ctx, "tunnel open", slog.String("hostname", hostname))

	return tunnel, nil
}

func terminableCommand(running context.Context, configuration Configuration) *exec.Cmd {
	//nolint:gosec // the arguments are the fixed tunnel policy and the configured tunnel
	command := exec.CommandContext(running, executableName, argumentsFor(configuration)...)
	command.Stderr = os.Stderr
	command.WaitDelay = terminationGrace
	command.Cancel = func() error {
		return command.Process.Signal(syscall.SIGTERM)
	}

	return command
}

func argumentsFor(configuration Configuration) []string {
	arguments := []string{
		"-T",
		"-o", "BatchMode=yes",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=60",
		"-o", "ServerAliveCountMax=3",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile=" + configuration.KnownHostsPath,
	}

	if configuration.IdentityFilePath != "" {
		arguments = append(arguments,
			"-i", configuration.IdentityFilePath,
			"-o", "IdentitiesOnly=yes",
		)
	}

	return append(arguments,
		"-R", forwardFor(configuration.IngressPort),
		destinationFor(configuration),
		"--",
		"--output", "json",
		"--no-inject-http-proxy-headers",
		"--proxy-protocol-header-version", "v2",
	)
}

func forwardFor(ingressPort int) string {
	return strconv.Itoa(PublicPort) +
		":" + loopbackAddress +
		":" + strconv.Itoa(ingressPort)
}

func destinationFor(configuration Configuration) string {
	if configuration.IdentityFilePath == "" {
		return keylessUser + "@" + configuration.Host
	}

	return configuration.Host
}

func eventLines(events io.Reader) *bufio.Scanner {
	lines := bufio.NewScanner(events)
	lines.Buffer(make([]byte, 0, maximumEventLineByte), maximumEventLineByte)

	return lines
}

func (tunnel *Tunnel) read(ctx context.Context, lines *bufio.Scanner) {
	assigned := ""

	for lines.Scan() {
		hostname, err := assignedHostnameFrom(lines.Bytes())
		switch {
		case errors.Is(err, errNotAForwardEvent):
			continue
		case err != nil:
			tunnel.fail(ctx, err)

			return
		case hostname == assigned:
			continue
		case assigned != "":
			tunnel.fail(ctx, fmt.Errorf("%w: hostname %s replaced by %s",
				ErrBadEvent, assigned, hostname))

			return
		}

		assigned = hostname
		tunnel.assigned <- hostname
	}

	if err := lines.Err(); err != nil {
		tunnel.fail(ctx, fmt.Errorf("read the tunnel events: %w", err))

		return
	}

	tunnel.fail(ctx, endedWith(tunnel.command.Wait()))
}

func (tunnel *Tunnel) fail(ctx context.Context, err error) {
	slog.WarnContext(ctx, "tunnel failed", slog.Any("error", err))

	select {
	case tunnel.failure <- err:
	default:
	}
}

func endedWith(waited error) error {
	if waited != nil {
		return fmt.Errorf("%w: %w", ErrTunnelEnded, waited)
	}

	return ErrTunnelEnded
}

func (tunnel *Tunnel) acquiredHostname(ctx context.Context) (string, error) {
	timer := time.NewTimer(acquisitionTimeout)
	defer timer.Stop()

	select {
	case hostname := <-tunnel.assigned:
		return hostname, nil
	case err := <-tunnel.failure:
		return "", fmt.Errorf("acquire the tunnel hostname: %w", err)
	case <-timer.C:
		return "", fmt.Errorf("acquire the tunnel hostname: %w", ErrNoAssignedHostname)
	case <-ctx.Done():
		return "", fmt.Errorf("acquire the tunnel hostname: %w", ctx.Err())
	}
}

func (tunnel *Tunnel) Hostname() string {
	return tunnel.hostname
}

func (tunnel *Tunnel) Failure() <-chan error {
	return tunnel.failure
}

func (tunnel *Tunnel) Close() error {
	tunnel.stop()

	return nil
}
