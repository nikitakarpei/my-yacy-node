package redirectresolution_test

import (
	"context"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redirectresolution"
)

func newBucket(t *testing.T) jetstream.KeyValue {
	t.Helper()
	srv, err := natsserver.NewServer(&natsserver.Options{
		Port: -1, JetStream: true, StoreDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("nats not ready")
	}
	t.Cleanup(srv.Shutdown)
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatal(err)
	}
	if err := yacycrawlcontract.EnsureRedirectResolutionBucket(
		context.Background(), js, yacycrawlcontract.RedirectResolutionBucketSpec{},
	); err != nil {
		t.Fatal(err)
	}
	bucket, err := js.KeyValue(context.Background(), yacycrawlcontract.RedirectResolutionBucketName)
	if err != nil {
		t.Fatal(err)
	}
	return bucket
}

func TestRecordWritesContractKeyWithCanonicalValue(t *testing.T) {
	bucket := newBucket(t)
	recorder := redirectresolution.New(bucket)

	const requested = "http://example.com/a"
	const canonical = "http://example.com/c"
	if err := recorder.Record(context.Background(), requested, canonical); err != nil {
		t.Fatalf("record: %v", err)
	}

	entry, err := bucket.Get(
		context.Background(),
		yacycrawlcontract.RedirectResolutionKey(requested),
	)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got := string(entry.Value()); got != canonical {
		t.Fatalf("value = %q, want %q", got, canonical)
	}
}

func TestRecordSkipsSelfEdge(t *testing.T) {
	bucket := newBucket(t)
	recorder := redirectresolution.New(bucket)

	const url = "http://example.com/a"
	if err := recorder.Record(context.Background(), url, url); err != nil {
		t.Fatalf("record: %v", err)
	}

	if _, err := bucket.Get(
		context.Background(),
		yacycrawlcontract.RedirectResolutionKey(url),
	); err == nil {
		t.Fatal("self-edge should not have been written")
	}
}
