package redisrecord_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redisrecord"
)

const retention = time.Hour

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

func talliesOn(server *miniredis.Miniredis) *redisrecord.Records[tally] {
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	return redisrecord.New[tally](client, retention)
}

func tallies(t *testing.T) *redisrecord.Records[tally] {
	t.Helper()
	return talliesOn(miniredis.RunT(t))
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

func TestARevisedRecordIsGoneOnceItsRetentionPasses(t *testing.T) {
	server := miniredis.RunT(t)
	records := talliesOn(server)
	ctx := context.Background()
	if _, _, err := records.Revise(ctx, "kept", countedTally); err != nil {
		t.Fatalf("revise: %v", err)
	}

	server.FastForward(retention)

	standing, err := records.RecordAt(ctx, "kept")
	if err != nil {
		t.Fatalf("read the expired record: %v", err)
	}
	if standing.Count != 0 {
		t.Fatalf("a record past its retention read as %d, want the zero record", standing.Count)
	}
}

func TestAFailureOfTheServerIsNotReportedAsContention(t *testing.T) {
	server := miniredis.RunT(t)
	records := talliesOn(server)
	server.Close()

	_, _, err := records.Revise(context.Background(), "orphaned", countedTally)
	if err == nil {
		t.Fatal("revising a record on a closed server should fail")
	}
	if errors.Is(err, redisrecord.ErrContended) {
		t.Fatalf("a closed server was reported as contention: %v", err)
	}
}

func TestKeysOfOneKindDifferByWhatTheyWereMadeFrom(t *testing.T) {
	first := redisrecord.KeyOf("KIND", "o1", "http://host/a")
	second := redisrecord.KeyOf("KIND", "o1", "http://host/b")
	otherKind := redisrecord.KeyOf("OTHER", "o1", "http://host/a")

	if first == second || first == otherKind {
		t.Fatalf("keys collide: %s %s %s", first, second, otherKind)
	}
}
