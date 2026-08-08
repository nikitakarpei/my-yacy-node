package jetstream_test

import (
	"context"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redirectresolvers/jetstream"
)

func newBucket(t *testing.T) natsjetstream.KeyValue {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateKeyValue(context.Background(), natsjetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.RedirectResolutionBucketName,
	}); err != nil {
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
	recorder := jetstream.New(bucket)

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
	recorder := jetstream.New(bucket)

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
