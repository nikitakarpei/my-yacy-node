// Package natstestserver starts an embedded NATS JetStream server for tests. A test can also start
// one with two users: one that may do anything, and one limited to the subjects the test allows.
package natstestserver

import (
	"net/url"
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

	unlimitedUser = "unlimited"
	limitedUser   = "limited"
	userPassword  = "test"
)

func Start(t *testing.T) string {
	t.Helper()
	return StartedServer(t).ClientURL()
}

func StartedServer(t *testing.T) *natsserver.Server {
	t.Helper()
	return startedServerWith(t, &natsserver.Options{})
}

func startedServerWith(t *testing.T, options *natsserver.Options) *natsserver.Server {
	t.Helper()
	options.Port = -1
	options.JetStream = true
	options.StoreDir = storeDirectoryFor(t)
	srv, err := natsserver.NewServer(options)
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
	return srv
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

type Allowed struct {
	Publish   []string
	Subscribe []string
}

type UserURLs struct {
	Unlimited string
	Limited   string
}

func StartWithLimitedUser(t *testing.T, allowed Allowed) UserURLs {
	t.Helper()
	srv := startedServerWith(t, &natsserver.Options{Users: []*natsserver.User{
		{Username: unlimitedUser, Password: userPassword},
		{Username: limitedUser, Password: userPassword, Permissions: &natsserver.Permissions{
			Publish:   &natsserver.SubjectPermission{Allow: allowed.Publish},
			Subscribe: &natsserver.SubjectPermission{Allow: allowed.Subscribe},
		}},
	}})
	return UserURLs{Unlimited: userURLOf(srv, unlimitedUser), Limited: userURLOf(srv, limitedUser)}
}

func userURLOf(srv *natsserver.Server, user string) string {
	address := srv.Addr().String()
	return (&url.URL{Scheme: "nats", User: url.UserPassword(user, userPassword), Host: address}).String()
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
