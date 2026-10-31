package rwiescrow_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const (
	holdFor             = 5 * time.Minute
	roomForEveryPosting = 100
)

var holdingOrigin = rwiorigin.Of("203.0.113.7")

type harness struct {
	engine vault.Engine
	vault  *vault.Vault
	escrow *rwiescrow.PostingEscrow
	clock  time.Time
}

func openHarness(t *testing.T) *harness {
	t.Helper()

	return openCappedHarness(t, roomForEveryPosting)
}

func openCappedHarness(t *testing.T, capacity int) *harness {
	t.Helper()

	engine := memoryvault.OpenEngine(0)
	v, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(engine),
		nil,
		nil,
		wallclock.Clock{},
	)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	h := &harness{
		engine: engine,
		vault:  v,
		clock:  time.Unix(1700000000, 123456789),
	}
	escrow, err := rwiescrow.Open(
		context.Background(),
		v,
		capacity,
		func() time.Time { return h.clock },
	)
	if err != nil {
		t.Fatalf("rwiescrow.Open: %v", err)
	}
	h.escrow = escrow

	return h
}

func (h *harness) holdUnproven(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	h.update(t, func(tx *vault.Txn) error {
		for _, entry := range postings {
			version := pageversion.FromPosting(entry, pageversion.SourcePeer(holdingOrigin))
			if err := h.escrow.HoldUnproven(tx, holdingOrigin, entry, version); err != nil {
				return fmt.Errorf("hold unproven: %w", err)
			}
		}

		return nil
	})
}

func (h *harness) holdProven(t *testing.T, provenPostings ...yacymodel.ProvenPosting) {
	t.Helper()

	h.update(t, func(tx *vault.Txn) error {
		for _, provenPosting := range provenPostings {
			version := pageversion.FromProvenPosting(
				provenPosting,
				pageversion.SourcePeer(holdingOrigin),
			)
			if err := h.escrow.HoldProven(tx, holdingOrigin, provenPosting, version); err != nil {
				return fmt.Errorf("hold proven: %w", err)
			}
		}

		return nil
	})
}

func (h *harness) update(t *testing.T, write func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), write); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h *harness) waitingFor(t *testing.T, url yacymodel.URLHash) rwiescrow.WaitingPostings {
	t.Helper()

	var waiting rwiescrow.WaitingPostings
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		postings, err := h.escrow.PostingsWaitingFor(context.Background(), tx, url)
		waiting = postings

		return err
	}); err != nil {
		t.Fatalf("PostingsWaitingFor: %v", err)
	}

	return waiting
}

func (h *harness) release(t *testing.T, postings ...yacymodel.RWIPosting) {
	t.Helper()

	var released []rwiescrow.WaitingPosting
	for _, entry := range postings {
		for _, waiting := range h.waitingFor(t, entry.URLHash).Unproven {
			if holdsOnly(waiting, entry) {
				released = append(released, waiting)
			}
		}
	}
	h.update(t, func(tx *vault.Txn) error {
		for _, waiting := range released {
			if err := h.escrow.ReleaseUnproven(tx, waiting); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *harness) releaseProven(t *testing.T, waiting rwiescrow.WaitingProvenPosting) {
	t.Helper()

	h.update(t, func(tx *vault.Txn) error {
		return h.escrow.ReleaseProven(tx, waiting)
	})
}

func (h *harness) escrowedCount(t *testing.T) int {
	t.Helper()

	var count int
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		measured, err := h.escrow.Count(tx)
		count = measured

		return err
	}); err != nil {
		t.Fatalf("Count: %v", err)
	}

	return count
}

func (h *harness) isFull(t *testing.T) bool {
	t.Helper()

	full, err := h.escrow.IsFull(context.Background())
	if err != nil {
		t.Fatalf("IsFull: %v", err)
	}

	return full
}

func urlHash(seed string) yacymodel.URLHash {
	address, err := url.Parse("http://example.com/" + seed)
	if err != nil {
		panic(err)
	}

	return yacymodel.URLHashOf(address)
}

func posting(word, urlSeed string) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:   yacymodel.WordHash(word),
		URLHash:    urlHash(urlSeed),
		Language:   yacymodel.LanguageOfUndeclaredDocument,
		LocalLinks: 1,
		Hits:       1,
	}
}

func TestHeldPostingWaitsForItsURLWithItsOrigin(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.holdUnproven(t, entry)

	waiting := h.waitingFor(t, entry.URLHash)
	if len(waiting.Unproven) != 1 || !holdsOnly(waiting.Unproven[0], entry) ||
		waiting.Unproven[0].Origin() != holdingOrigin || len(waiting.Proven) != 0 {
		t.Fatalf("waiting = %+v, want the held posting with its origin", waiting)
	}
}

func TestPostingsWaitingForAURLLeaveOtherURLsOut(t *testing.T) {
	h := openHarness(t)
	h.holdUnproven(t, posting("w1", "u1"), posting("w2", "u1"), posting("w1", "u2"))

	if waiting := h.waitingFor(t, urlHash("u1")); len(waiting.Unproven) != 2 {
		t.Fatalf("waiting = %+v, want the two postings of u1", waiting)
	}
}

func TestReleasedPostingStopsWaiting(t *testing.T) {
	h := openHarness(t)
	released, kept := posting("w1", "u1"), posting("w2", "u1")
	h.holdUnproven(t, released, kept)

	h.release(t, released)

	waiting := h.waitingFor(t, urlHash("u1"))
	if len(waiting.Unproven) != 1 || !holdsOnly(waiting.Unproven[0], kept) {
		t.Fatalf("waiting = %+v, want only the kept posting", waiting)
	}
}

func TestEscrowedPostingExpiresAfterHoldPeriod(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.holdUnproven(t, entry)

	expired, err := h.escrow.Expire(ctx, holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 0 {
		t.Fatalf("expired = %d within the hold period, want 0", expired)
	}

	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err = h.escrow.Expire(ctx, holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d after the hold period, want 1", expired)
	}
	if got := h.escrowedCount(t); got != 0 {
		t.Fatalf("escrowed count = %d after expiry, want 0", got)
	}
}

func TestReHoldRefreshesTheHoldInsteadOfDuplicatingIt(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.holdUnproven(t, entry)
	h.clock = h.clock.Add(holdFor / 2)
	h.holdUnproven(t, entry)

	if got := h.escrowedCount(t); got != 1 {
		t.Fatalf("escrowed count = %d, want 1", got)
	}

	h.clock = h.clock.Add(holdFor/2 + time.Second)
	expired, err := h.escrow.Expire(context.Background(), holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 0 {
		t.Fatalf(
			"expired = %d at the original hold's deadline, want 0 since the hold was refreshed",
			expired,
		)
	}

	h.clock = h.clock.Add(holdFor)
	expired, err = h.escrow.Expire(context.Background(), holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d after the refreshed hold's deadline, want 1", expired)
	}
}

const cappedCapacity = 3

func TestEscrowIsFullOnceItHoldsTheCapacity(t *testing.T) {
	h := openCappedHarness(t, cappedCapacity)
	admitted := postingsNumbering(cappedCapacity)
	h.holdUnproven(t, admitted[:cappedCapacity-1]...)
	if h.isFull(t) {
		t.Fatalf("escrow below its capacity reports it is full")
	}

	h.holdUnproven(t, admitted[cappedCapacity-1])

	if !h.isFull(t) {
		t.Fatalf("escrow at its capacity reports room")
	}
}

func TestHoldKeepsAPostingBeyondTheCapacity(t *testing.T) {
	h := openCappedHarness(t, cappedCapacity)
	h.holdUnproven(t, postingsNumbering(cappedCapacity)...)

	h.holdUnproven(t, posting("w1", "beyond"))

	if got := h.escrowedCount(t); got != cappedCapacity+1 {
		t.Fatalf("escrowed count = %d, want %d past the capacity", got, cappedCapacity+1)
	}
}

func postingsNumbering(postings int) []yacymodel.RWIPosting {
	numbered := make([]yacymodel.RWIPosting, postings)
	for position := range numbered {
		numbered[position] = posting("w1", fmt.Sprintf("u%d", position))
	}

	return numbered
}

func TestExpireStopsAtLimit(t *testing.T) {
	h := openHarness(t)
	h.holdUnproven(t, posting("w1", "u1"), posting("w1", "u2"), posting("w1", "u3"))
	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err := h.escrow.Expire(context.Background(), holdFor, 2)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 2 {
		t.Fatalf("expired = %d, want 2 at the limit", expired)
	}
	if got := h.escrowedCount(t); got != 1 {
		t.Fatalf("escrowed count = %d, want 1 left for the next run", got)
	}
}

func TestExpireTakesTheOldestHoldFirst(t *testing.T) {
	ctx := context.Background()
	h := openHarness(t)
	oldest, newest := posting("w1", "u1"), posting("w1", "u2")

	h.holdUnproven(t, oldest)
	h.clock = h.clock.Add(time.Minute)
	h.holdUnproven(t, newest)
	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err := h.escrow.Expire(ctx, holdFor, 1)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d, want 1 at the limit", expired)
	}

	if len(h.waitingFor(t, oldest.URLHash).Unproven) != 0 {
		t.Fatal("the oldest hold outlived an expiry run that stopped at one posting")
	}
	if len(h.waitingFor(t, newest.URLHash).Unproven) != 1 {
		t.Fatal("the newer hold expired before the oldest one")
	}
}

func TestReleasedPostingIsNotExpired(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")

	h.holdUnproven(t, entry)
	h.release(t, entry)
	h.clock = h.clock.Add(holdFor + time.Second)

	expired, err := h.escrow.Expire(context.Background(), holdFor, 10)
	if err != nil {
		t.Fatalf("Expire: %v", err)
	}
	if expired != 0 {
		t.Fatalf("expired = %d, want 0 after release dropped both rows", expired)
	}
}

func TestEscrowHoldsEachVersionOfAWordApart(t *testing.T) {
	h := openHarness(t)
	newer, older := posting("w1", "u1"), posting("w1", "u1")
	newer.LastModified = 20000
	older.LastModified = 19000

	h.holdUnproven(t, newer, older)
	h.release(t, older)

	waiting := h.waitingFor(t, newer.URLHash)
	if len(waiting.Unproven) != 1 || !holdsOnly(waiting.Unproven[0], newer) {
		t.Fatalf("waiting = %+v, want the newer version still waiting", waiting)
	}
}

func holdsOnly(waiting rwiescrow.WaitingPosting, posting yacymodel.RWIPosting) bool {
	return waiting.Posting() == posting
}

const crawlerRingExponent = 2

func crawledProvenPostingsFrom(postings ...yacymodel.RWIPosting) []yacymodel.ProvenPosting {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(crawlerRingExponent)
	if err != nil {
		panic(err)
	}
	crawled, err := yacymodel.ProvenPagePostingsFrom(
		postings,
		yacymodel.EmptyPostingsProof(partitions),
	)
	if err != nil {
		panic(err)
	}

	return crawled.EachProvenPosting()
}

func TestHeldPostingWaitsWithTheProofItArrivedWith(t *testing.T) {
	h := openHarness(t)
	proven := crawledProvenPostingsFrom(posting("w1", "u1"), posting("w2", "u1"))[0]

	h.holdProven(t, proven)

	waiting := h.waitingFor(t, urlHash("u1"))
	if len(waiting.Proven) != 1 || len(waiting.Unproven) != 0 ||
		waiting.Proven[0].ProvenPosting().Posting() != proven.Posting() ||
		waiting.Proven[0].Origin() != holdingOrigin {
		t.Fatalf("waiting = %+v, want the proven posting with its origin", waiting)
	}
	if waitingProven := waiting.Proven[0].ProvenPosting(); waitingProven.Root() != proven.Root() {
		t.Fatalf("waiting proven posting = %+v, want the root it arrived under %+v",
			waitingProven, proven.Root())
	}
}

func TestEscrowHoldsEachRootOfAWordApart(t *testing.T) {
	h := openHarness(t)
	entry := posting("w1", "u1")
	underOneRoot := crawledProvenPostingsFrom(entry)[0]
	underAnotherRoot := crawledProvenPostingsFrom(entry, posting("w2", "u1"))[0]

	h.holdProven(t, underOneRoot, underAnotherRoot)
	h.holdUnproven(t, entry)
	for _, waiting := range h.waitingFor(t, entry.URLHash).Proven {
		if waiting.ProvenPosting().Root() == underAnotherRoot.Root() {
			h.releaseProven(t, waiting)
		}
	}

	waiting := h.waitingFor(t, entry.URLHash)
	if len(waiting.Proven) != 1 || len(waiting.Unproven) != 1 || h.escrowedCount(t) != 2 {
		t.Fatalf("waiting = %+v, want the posting under one root and the unproven one", waiting)
	}
	if waiting.Proven[0].ProvenPosting().Root() != underOneRoot.Root() {
		t.Fatalf("waiting = %+v, want the released root gone", waiting)
	}
}

func TestPostingsWaitingForAURLLeaveOutAPostingWithARefusedProof(t *testing.T) {
	h := openHarness(t)
	h.holdProven(t, crawledProvenPostingsFrom(posting("w1", "u1"), posting("w2", "u1"))...)

	h.swapStoredValuesOf(t, "rwi_escrow")

	if waiting := h.waitingFor(t, urlHash("u1")); len(waiting.Proven) != 0 {
		t.Fatalf("waiting = %+v, want no posting with a refused escrowed proof", waiting)
	}
}

func (h *harness) swapStoredValuesOf(t *testing.T, bucket vault.Name) {
	t.Helper()

	err := h.engine.Update(t.Context(), func(tx vault.EngineTxn) error {
		records := tx.Records(bucket)
		var keys, values [][]byte
		if err := records.Scan(vault.EveryKey(), func(key, value []byte) (bool, error) {
			keys, values = append(keys, key), append(values, value)

			return true, nil
		}); err != nil {
			return err
		}
		for index, key := range keys {
			if _, err := records.Put(key, values[len(values)-1-index]); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("swap stored values: %v", err)
	}
}
