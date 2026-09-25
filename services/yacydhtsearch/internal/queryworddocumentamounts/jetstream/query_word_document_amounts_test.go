package jetstream_test

import (
	"context"
	"maps"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	queryworddocumentamountsjetstream "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryworddocumentamounts/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const bucketName = "query-word-document-amounts"

type recordedFailures struct {
	lookups []yacymodel.Hash
	stores  []yacymodel.Hash
}

func (r *recordedFailures) DocumentAmountLookupFailed(
	_ context.Context,
	word yacymodel.Hash,
	_ error,
) {
	r.lookups = append(r.lookups, word)
}

func (r *recordedFailures) DocumentAmountStoreFailed(
	_ context.Context,
	word yacymodel.Hash,
	_ error,
) {
	r.stores = append(r.stores, word)
}

type bucketOnATestServer struct {
	stream natsjetstream.JetStream
	bucket natsjetstream.KeyValue
}

func bucketOnATestServerFor(t *testing.T) bucketOnATestServer {
	t.Helper()

	stream := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	bucket, err := stream.CreateOrUpdateKeyValue(
		t.Context(), natsjetstream.KeyValueConfig{Bucket: bucketName},
	)
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}

	return bucketOnATestServer{stream: stream, bucket: bucket}
}

func (server bucketOnATestServer) deleteTheBucket(t *testing.T) {
	t.Helper()

	if err := server.stream.DeleteKeyValue(t.Context(), bucketName); err != nil {
		t.Fatalf("delete bucket: %v", err)
	}
}

func TestAnAmountOneInstanceRemembersIsReadBackByAnother(t *testing.T) {
	t.Parallel()

	server := bucketOnATestServerFor(t)
	failures := &recordedFailures{}
	observers := queryworddocumentamountsjetstream.QueryWordDocumentAmountsObservers{failures}
	berlin, weather := yacymodel.WordHash("berlin"), yacymodel.WordHash("weather")
	queryworddocumentamountsjetstream.New(server.bucket, observers).
		Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})

	got := queryworddocumentamountsjetstream.New(server.bucket, observers).
		DocumentAmountsOf(t.Context(), []yacymodel.Hash{berlin, weather})

	if want := map[yacymodel.Hash]int{berlin: 12}; !maps.Equal(got, want) {
		t.Fatalf("DocumentAmountsOf = %v, want %v", got, want)
	}
	if len(failures.lookups) != 0 || len(failures.stores) != 0 {
		t.Fatalf("failures reported %+v, want none", failures)
	}
}

func TestALookupTheBucketRefusesIsReportedAndIsAMiss(t *testing.T) {
	t.Parallel()

	server := bucketOnATestServerFor(t)
	failures := &recordedFailures{}
	amounts := queryworddocumentamountsjetstream.New(
		server.bucket,
		queryworddocumentamountsjetstream.QueryWordDocumentAmountsObservers{failures},
	)
	berlin := yacymodel.WordHash("berlin")
	amounts.Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})
	server.deleteTheBucket(t)

	got := amounts.DocumentAmountsOf(t.Context(), []yacymodel.Hash{berlin})

	if len(got) != 0 || len(failures.lookups) != 1 || failures.lookups[0] != berlin {
		t.Fatalf("DocumentAmountsOf = %v with lookup failures %v, want a miss reported for berlin",
			got, failures.lookups)
	}
}

func TestAnAmountTheBucketRefusesToStoreIsReported(t *testing.T) {
	t.Parallel()

	server := bucketOnATestServerFor(t)
	failures := &recordedFailures{}
	amounts := queryworddocumentamountsjetstream.New(
		server.bucket,
		queryworddocumentamountsjetstream.QueryWordDocumentAmountsObservers{failures},
	)
	server.deleteTheBucket(t)
	berlin := yacymodel.WordHash("berlin")

	amounts.Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})

	if len(failures.stores) != 1 || failures.stores[0] != berlin {
		t.Fatalf("store failures = %v, want one for berlin", failures.stores)
	}
}

func TestAValueThatIsNotAnAmountIsReportedAndIsAMiss(t *testing.T) {
	t.Parallel()

	server := bucketOnATestServerFor(t)
	failures := &recordedFailures{}
	berlin := yacymodel.WordHash("berlin")
	if _, err := server.bucket.Put(t.Context(), berlin.String(), []byte("many")); err != nil {
		t.Fatalf("put: %v", err)
	}

	got := queryworddocumentamountsjetstream.New(
		server.bucket,
		queryworddocumentamountsjetstream.QueryWordDocumentAmountsObservers{failures},
	).DocumentAmountsOf(t.Context(), []yacymodel.Hash{berlin})

	if len(got) != 0 || len(failures.lookups) != 1 {
		t.Fatalf("DocumentAmountsOf = %v with lookup failures %v, want a reported miss",
			got, failures.lookups)
	}
}
