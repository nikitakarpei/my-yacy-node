// Package processenvironmentleaseserver owns the Unix socket that carries one
// process environment lease to the node. It closes a consumer that connects
// before the grant is authoritative, writes the single grant frame to the first
// consumer that connects afterwards, and reports the end of that consumer.
package processenvironmentleaseserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/processenvironmentlease"
)

const (
	socketNetwork        = "unix"
	socketDirectoryMode  = 0o770
	socketMode           = 0o660
	frameWriteTimeout    = 5 * time.Second
	stalePathDialTimeout = time.Second
)

var (
	ErrSocketInUse   = errors.New("the lease socket has an active listener")
	ErrConsumerEnded = errors.New("the lease consumer ended")
)

type Server struct {
	listener        net.Listener
	granted         chan processenvironmentlease.Grant
	failure         chan error
	stopListeningOn sync.Once
}

func Listen(ctx context.Context, socketPath string) (*Server, error) {
	if err := os.MkdirAll(filepath.Dir(socketPath), socketDirectoryMode); err != nil {
		return nil, fmt.Errorf("make the lease socket directory: %w", err)
	}

	if err := unlinkStalePath(ctx, socketPath); err != nil {
		return nil, err
	}

	listenConfiguration := net.ListenConfig{}

	listener, err := listenConfiguration.Listen(ctx, socketNetwork, socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on the lease socket: %w", err)
	}

	if err := os.Chmod(socketPath, socketMode); err != nil {
		return nil, fmt.Errorf("open the lease socket to the consumer: %w", err)
	}

	server := &Server{
		listener: listener,
		granted:  make(chan processenvironmentlease.Grant, 1),
		failure:  make(chan error, 1),
	}
	go server.accept(context.WithoutCancel(ctx))

	slog.DebugContext(ctx, "lease socket listening", slog.String("socketPath", socketPath))

	return server, nil
}

func unlinkStalePath(ctx context.Context, socketPath string) error {
	if _, err := os.Stat(socketPath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	dialer := net.Dialer{Timeout: stalePathDialTimeout}

	connection, err := dialer.DialContext(ctx, socketNetwork, socketPath)
	if err == nil {
		defer closeConnection(ctx, connection)

		return fmt.Errorf("%w: %s", ErrSocketInUse, socketPath)
	}

	if err := os.Remove(socketPath); err != nil {
		return fmt.Errorf("unlink the stale lease socket: %w", err)
	}

	return nil
}

func (server *Server) accept(ctx context.Context) {
	for {
		connection, err := server.listener.Accept()
		if err != nil {
			server.fail(ctx, fmt.Errorf("accept a lease consumer: %w", err))

			return
		}

		select {
		case grant := <-server.granted:
			server.stopListening(ctx)
			server.serve(ctx, connection, grant)

			return
		default:
			slog.WarnContext(ctx, "lease consumer connected before the grant")
			closeConnection(ctx, connection)
		}
	}
}

func (server *Server) serve(
	ctx context.Context,
	connection net.Conn,
	grant processenvironmentlease.Grant,
) {
	defer closeConnection(ctx, connection)

	frame, err := processenvironmentlease.FrameOf(grant)
	if err != nil {
		server.fail(ctx, fmt.Errorf("form the grant frame: %w", err))

		return
	}

	if err := connection.SetWriteDeadline(time.Now().Add(frameWriteTimeout)); err != nil {
		server.fail(ctx, fmt.Errorf("write the grant frame: %w", err))

		return
	}

	if _, err := connection.Write(frame); err != nil {
		server.fail(ctx, fmt.Errorf("write the grant frame: %w", err))

		return
	}

	slog.DebugContext(ctx, "process environment lease granted")

	server.fail(ctx, consumerEndedWith(waitForConsumerEnd(connection)))
}

func waitForConsumerEnd(connection net.Conn) error {
	trailing := make([]byte, 1)

	if _, err := connection.Read(trailing); err != nil {
		return fmt.Errorf("read from the consumer: %w", err)
	}

	return fmt.Errorf("byte %q from the consumer", trailing)
}

func consumerEndedWith(read error) error {
	if errors.Is(read, io.EOF) {
		return ErrConsumerEnded
	}

	return fmt.Errorf("%w: %w", ErrConsumerEnded, read)
}

func (server *Server) fail(ctx context.Context, err error) {
	slog.WarnContext(ctx, "process environment lease ended", slog.Any("error", err))

	select {
	case server.failure <- err:
	default:
	}
}

func (server *Server) stopListening(ctx context.Context) {
	server.stopListeningOn.Do(func() {
		if err := server.listener.Close(); err != nil {
			slog.WarnContext(ctx, "lease socket not closed", slog.Any("error", err))
		}
	})
}

func closeConnection(ctx context.Context, connection net.Conn) {
	if err := connection.Close(); err != nil {
		slog.WarnContext(ctx, "lease connection not closed", slog.Any("error", err))
	}
}

func (server *Server) Grant(grant processenvironmentlease.Grant) {
	server.granted <- grant
}

func (server *Server) Failure() <-chan error {
	return server.failure
}

func (server *Server) Close() error {
	server.stopListening(context.Background())

	return nil
}
