package jetstream_test

import (
	"context"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/disposedpages/jetstream"
)

func TestRecordWritesContractKey(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateKeyValue(context.Background(), natsjetstream.KeyValueConfig{
		Bucket: yacycrawlcontract.DisposedPagesBucketName,
	}); err != nil {
		t.Fatal(err)
	}
	bucket, err := js.KeyValue(context.Background(), yacycrawlcontract.DisposedPagesBucketName)
	if err != nil {
		t.Fatal(err)
	}
	recorder := jetstream.New(bucket)

	url := canonicalurltest.CanonicalURLOf(t, "http://example.com/a")
	if err := recorder.Record(context.Background(), url); err != nil {
		t.Fatalf("record: %v", err)
	}

	if _, err := bucket.Get(
		context.Background(),
		yacycrawlcontract.DisposedPageKey(url),
	); err != nil {
		t.Fatalf("get: %v", err)
	}
}
