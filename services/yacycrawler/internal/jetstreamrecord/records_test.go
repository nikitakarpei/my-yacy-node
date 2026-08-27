package jetstreamrecord_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const bucketName = "RECORDS"

type tally struct {
	Count int
}

func countedTally(counted tally) (tally, bool) {
	counted.Count++
	return counted, true
}

func refusedTally(counted tally) (tally, bool) {
	return tally{Count: counted.Count + 1}, false
}

func tallies(t *testing.T) *jetstreamrecord.Records[tally] {
	t.Helper()
	return jetstreamrecord.New[tally](bucket(t))
}

func bucket(t *testing.T) natsjetstream.KeyValue {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	opened, err := js.CreateKeyValue(context.Background(), natsjetstream.KeyValueConfig{
		Bucket: bucketName,
	})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	return opened
}

func TestARecordNobodyWroteArrivesAsTheZeroRecord(t *testing.T) {
	records := tallies(t)
	ctx := context.Background()

	standing, err := records.RecordAt(ctx, "absent")
	if err != nil {
		t.Fatalf("read the absent record: %v", err)
	}
	if standing.Count != 0 {
		t.Fatalf("an absent record read as %d, want the zero record", standing.Count)
	}

	revised, wrote, err := records.Revise(ctx, "absent", countedTally)
	if err != nil || !wrote || revised.Count != 1 {
		t.Fatalf("revising an absent record gave %v %v %v, want 1 true nil", revised, wrote, err)
	}
}

func TestEveryConcurrentRevisionOfOneRecordLands(t *testing.T) {
	records := tallies(t)
	const writers = 6

	var writing sync.WaitGroup
	failures := make(chan error, writers)
	for range writers {
		writing.Add(1)
		go func() {
			defer writing.Done()
			if _, _, err := records.Revise(
				context.Background(),
				"shared",
				countedTally,
			); err != nil {
				failures <- err
			}
		}()
	}
	writing.Wait()
	close(failures)
	for err := range failures {
		t.Fatalf("a concurrent revision failed: %v", err)
	}

	standing, err := records.RecordAt(context.Background(), "shared")
	if err != nil {
		t.Fatalf("read the shared record: %v", err)
	}
	if standing.Count != writers {
		t.Fatalf("the record counted %d revisions, want %d", standing.Count, writers)
	}
}

func TestARefusedRevisionLeavesTheRecordUntouched(t *testing.T) {
	records := tallies(t)
	ctx := context.Background()
	if _, _, err := records.Revise(ctx, "refused", countedTally); err != nil {
		t.Fatalf("revise: %v", err)
	}

	standing, wrote, err := records.Revise(ctx, "refused", refusedTally)
	if err != nil {
		t.Fatalf("revise: %v", err)
	}
	if wrote {
		t.Fatal("a refused revision should not be written")
	}
	if standing.Count != 1 {
		t.Fatalf("a refused revision reported %d, want the standing 1", standing.Count)
	}

	stored, err := records.RecordAt(ctx, "refused")
	if err != nil {
		t.Fatalf("read the refused record: %v", err)
	}
	if stored.Count != 1 {
		t.Fatalf("a refused revision left %d stored, want 1", stored.Count)
	}
}

func TestAFailureOfTheBucketIsNotReportedAsContention(t *testing.T) {
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	opened, err := js.CreateKeyValue(ctx, natsjetstream.KeyValueConfig{Bucket: bucketName})
	if err != nil {
		t.Fatalf("create bucket: %v", err)
	}
	records := jetstreamrecord.New[tally](opened)
	if err := js.DeleteKeyValue(ctx, bucketName); err != nil {
		t.Fatalf("delete bucket: %v", err)
	}

	_, _, err = records.Revise(ctx, "orphaned", countedTally)
	if err == nil {
		t.Fatal("revising a record of a deleted bucket should fail")
	}
	if errors.Is(err, jetstreamrecord.ErrContended) {
		t.Fatalf("a deleted bucket was reported as contention: %v", err)
	}
}
