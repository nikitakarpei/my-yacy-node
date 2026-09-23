// Package natstestserver starts an embedded NATS JetStream server for tests.
package natstestserver

import (
	"os"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	unreachableServerURL = "nats://127.0.0.1:1"

	storeRemovalWait       = 5 * time.Second
	storeRemovalRetryDelay = 10 * time.Millisecond
)

func Start(t *testing.T) string {
	t.Helper()
	srv, err := natsserver.NewServer(&natsserver.Options{
		Port:      -1,
		JetStream: true,
		StoreDir:  storeDirectoryFor(t),
	})
	if err != nil {
		t.Fatalf("new nats server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("nats server not ready")
	}
	t.Cleanup(func() {
		srv.Shutdown()
		srv.WaitForShutdown()
	})
	return srv.ClientURL()
}

func storeDirectoryFor(t *testing.T) string {
	t.Helper()
	directory, err := os.MkdirTemp("", "natstestserver")
	if err != nil {
		t.Fatalf("create the jetstream store directory: %v", err)
	}
	t.Cleanup(func() { removeStoreDirectory(t, directory) })
	return directory
}

func removeStoreDirectory(t *testing.T, directory string) {
	t.Helper()
	deadline := time.Now().Add(storeRemovalWait)
	for {
		err := os.RemoveAll(directory)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("remove the jetstream store directory: %v", err)
			return
		}
		time.Sleep(storeRemovalRetryDelay)
	}
}

func ConnectWithoutServer(t *testing.T) *nats.Conn {
	t.Helper()
	connection, err := nats.Connect(
		unreachableServerURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Hour),
	)
	if err != nil {
		t.Fatalf("connect nats without a server: %v", err)
	}
	t.Cleanup(connection.Close)

	return connection
}

func ConnectJetStream(t *testing.T, url string) jetstream.JetStream {
	t.Helper()
	js, err := jetstream.New(Connect(t, url))
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return js
}

func Connect(t *testing.T, url string) *nats.Conn {
	t.Helper()
	connection, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(connection.Close)
	return connection
}
