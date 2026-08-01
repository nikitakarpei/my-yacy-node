package rwidistribution

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type fakeCourier struct {
	receipts map[yacymodel.Hash]postingOfferReceipt
	offered  []postingOffer
	onOffer  func()
}

func (f *fakeCourier) Offer(_ context.Context, _ string, offer postingOffer) postingOfferReceipt {
	f.offered = append(f.offered, offer)
	if f.onOffer != nil {
		f.onOffer()
	}

	return f.receipts[offer.Peer.Hash]
}

type fakeURLDirectory struct {
	metadata map[yacymodel.URLHash]yacymodel.URLMetadata
}

func (f fakeURLDirectory) MetadataByHash(
	_ context.Context,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLMetadata, error) {
	found := make([]yacymodel.URLMetadata, 0, len(hashes))
	for _, hash := range hashes {
		if entry, ok := f.metadata[hash]; ok {
			found = append(found, entry)
		}
	}

	return found, nil
}

func (f fakeURLDirectory) MissingURLs(
	_ context.Context,
	hashes []yacymodel.URLHash,
) ([]yacymodel.URLHash, error) {
	missing := make([]yacymodel.URLHash, 0, len(hashes))
	for _, hash := range hashes {
		if _, ok := f.metadata[hash]; !ok {
			missing = append(missing, hash)
		}
	}

	return missing, nil
}

func (fakeURLDirectory) Count(context.Context) (int, error) { return 0, nil }

type fakeURLMetadataCourier struct {
	receipt   urlMetadataReceipt
	delivered []yacymodel.URLMetadata
}

func (f *fakeURLMetadataCourier) Deliver(
	_ context.Context,
	_ string,
	_ yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) urlMetadataReceipt {
	f.delivered = append(f.delivered, metadata...)

	return f.receipt
}

func openPostingOfferCycle(
	t *testing.T,
	now func() time.Time,
	postings map[yacymodel.Hash]yacymodel.RWIPosting,
	responsible []yacymodel.Seed,
) (*postingOfferSchedule, *replicaLedger, *fakeCourier, *fakePostingOfferCycleObserver, *postingOfferCycle) {
	t.Helper()

	schedule, ledger, reader := openPostingReplicationReader(t, now, postings, responsible)
	courier := &fakeCourier{receipts: make(map[yacymodel.Hash]postingOfferReceipt)}
	observer := newFakePostingOfferCycleObserver()

	cycle := &postingOfferCycle{
		reader: reader,
		delivery: &postingOfferDelivery{
			postingCourier: courier,
			urlMetadataCourier: &fakeURLMetadataCourier{
				receipt: urlMetadataReceipt{Outcome: urlMetadataAccepted},
			},
			urls:     fakeURLDirectory{},
			observer: observer,
		},
		settlement: &postingOfferSettlement{
			ledger:   ledger,
			schedule: schedule,
			cadence: postingOfferCadence{
				refresh: time.Hour,
				retry:   time.Minute,
			},
			observer:   observer,
			now:        now,
			redundancy: postingReplicationReaderRedundancy,
		},
		schedule:         schedule,
		roster:           reader.roster,
		observer:         observer,
		now:              now,
		postingsPerCycle: 10,
		cycleInterval:    time.Minute,
	}

	return schedule, ledger, courier, observer, cycle
}

func TestPostingOfferCycleRunOffersOnceThenStops(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{Outcome: postingOfferAccepted}

	store(t, schedule, word, url)

	ctx, cancel := context.WithCancel(context.Background())
	courier.onOffer = cancel
	cycle.Run(ctx)

	if len(courier.offered) != 1 {
		t.Fatalf("offered = %v, want a single offer from the initial run", courier.offered)
	}
	if observer.due != 1 {
		t.Fatalf("due = %v, want 1", observer.due)
	}
}

func TestPostingOfferCycleSkipsWhenTooFewPeersReachable(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	cycle.minReachablePeers = 2

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	if len(courier.offered) != 0 {
		t.Fatalf(
			"offered = %v, want no offers while below the reachable-peer floor",
			courier.offered,
		)
	}
	if observer.cyclesSkipped != 1 {
		t.Fatalf("cyclesSkipped = %v, want 1", observer.cyclesSkipped)
	}

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] left untouched by a skipped cycle", due)
	}
}

func TestPostingOfferCycleReportsBacklogAgeOnSkippedCycle(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	cycle.minReachablePeers = 2

	store(t, schedule, word, url)

	later := now.Add(90 * time.Second)
	cycle.now = func() time.Time { return later }
	schedule.now = cycle.now
	cycle.runCycle(context.Background())

	if len(courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers on a skipped cycle", courier.offered)
	}
	if observer.oldestDueAge != 90*time.Second {
		t.Fatalf("oldestDueAge = %v, want 90s", observer.oldestDueAge)
	}
}

func TestPostingOfferCycleDropsStaleReplicaFromLedger(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	stalePeer, fresh := yacymodel.WordHash("stale"), yacymodel.WordHash("fresh")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(fresh)},
	)
	courier.receipts[fresh] = postingOfferReceipt{Outcome: postingOfferAccepted}

	store(t, schedule, word, url)
	if err := ledger.RecordAccepted(
		context.Background(),
		postingOffer{
			Peer:     seed(stalePeer),
			Postings: []yacymodel.RWIPosting{fakePosting(word, url)},
		},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	cycle.runCycle(context.Background())

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	for _, replica := range replicas {
		if replica == stalePeer {
			t.Fatalf("replicas = %v, want %v dropped", replicas, stalePeer)
		}
	}
	if observer.prunes != 1 {
		t.Fatalf("prunes = %v, want 1", observer.prunes)
	}
}

func TestPostingOfferCycleReschedulesAcceptedPostingAtRefreshInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{Outcome: postingOfferAccepted}

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due right after a fully replicated posting", due)
	}
}

func TestPostingOfferCycleRetriesRejectedPostingAtRetryInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{Outcome: postingOfferRefused}

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after a rejected offer", due)
	}

	future := now.Add(cycle.settlement.cadence.retry + time.Second)
	schedule.now = func() time.Time { return future }
	due, err = schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after retry interval: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the retry interval has elapsed", due)
	}
}

func TestPostingOfferCycleHonoursCourierRetryAfter(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{
		Outcome:    postingOfferDeferred,
		RetryAfter: 5 * time.Minute,
	}

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	afterCycleRetry := now.Add(cycle.settlement.cadence.retry + time.Second)
	schedule.now = func() time.Time { return afterCycleRetry }
	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the peer's pause elapses", due)
	}

	afterPause := now.Add(5*time.Minute + time.Second)
	schedule.now = func() time.Time { return afterPause }
	due, err = schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after pause: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the peer's pause has elapsed", due)
	}
}

func TestPostingOfferCycleReschedulesUnofferedPostingAtRetryInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openPostingOfferCycle(
		t,
		func() time.Time { return now },
		postings,
		nil,
	)

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	if len(courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers for an unoffered posting", courier.offered)
	}

	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due immediately after stalling", due)
	}

	future := now.Add(cycle.settlement.cadence.retry + time.Second)
	schedule.now = func() time.Time { return future }
	due, err = schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after retry interval: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the retry interval has elapsed", due)
	}
}

func TestPostingOfferCycleReschedulesAlreadySatisfiedPostingAtRefreshInterval(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, _, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)

	store(t, schedule, word, url)
	if err := ledger.RecordAccepted(
		context.Background(),
		postingOffer{Peer: seed(peer), Postings: []yacymodel.RWIPosting{fakePosting(word, url)}},
	); err != nil {
		t.Fatalf("RecordAccepted: %v", err)
	}

	cycle.runCycle(context.Background())

	if len(courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers for an already replicated posting", courier.offered)
	}

	future := now.Add(cycle.settlement.cadence.refresh - time.Second)
	schedule.now = func() time.Time { return future }
	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the refresh interval elapses", due)
	}
}

func TestPostingOfferCycleRecordsReplicaOnAcceptedOffer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{Outcome: postingOfferAccepted}

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 1 || replicas[0] != peer {
		t.Fatalf("replicas = %v, want [%v]", replicas, peer)
	}
	if observer.postingsOffered[string(postingOfferAccepted)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for outcome %q",
			observer.postingsOffered, postingOfferAccepted,
		)
	}
}

func TestPostingOfferCycleReschedulesAtMaxRetryAfterAcrossPeers(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peerA, peerB := yacymodel.WordHash("a"), yacymodel.WordHash("b")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, _, courier, _, cycle := openPostingOfferCycle(
		t,
		func() time.Time { return now },
		postings,
		[]yacymodel.Seed{seed(peerA), seed(peerB)},
	)
	cycle.reader.redundancy = 2
	cycle.settlement.redundancy = 2
	courier.receipts[peerA] = postingOfferReceipt{
		Outcome:    postingOfferDeferred,
		RetryAfter: time.Minute,
	}
	courier.receipts[peerB] = postingOfferReceipt{
		Outcome:    postingOfferDeferred,
		RetryAfter: 5 * time.Minute,
	}

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	afterShorterPause := now.Add(time.Minute + time.Second)
	schedule.now = func() time.Time { return afterShorterPause }
	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("due = %v, want none due before the longer pause elapses", due)
	}

	afterLongerPause := now.Add(5*time.Minute + time.Second)
	schedule.now = func() time.Time { return afterLongerPause }
	due, err = schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings after longer pause: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf("due = %v, want [word] once the longer pause has elapsed", due)
	}
}

func TestPostingOfferCycleCountsGonePostingSeparatelyFromDue(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	schedule, _, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, nil, []yacymodel.Seed{seed(peer)},
	)

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	if observer.due != 0 || observer.gone != 1 {
		t.Fatalf("due = %v, gone = %v, want 0 due and 1 gone", observer.due, observer.gone)
	}
	if len(courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offers for a posting gone from the index", courier.offered)
	}
}

func TestPostingOfferCycleRecordsNoReplicaOnRefusedOffer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{Outcome: postingOfferRefused}

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none after a refused offer", replicas)
	}
	if observer.postingsOffered[string(postingOfferRefused)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for outcome %q",
			observer.postingsOffered, postingOfferRefused,
		)
	}
}

func TestPostingOfferCycleReportsUnaddressablePeer(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	unaddressable := yacymodel.Seed{Hash: peer}
	schedule, ledger, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{unaddressable},
	)

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	if len(courier.offered) != 0 {
		t.Fatalf("offered = %v, want no offer sent to an unaddressable peer", courier.offered)
	}
	if observer.postingsOffered[string(postingOfferUnaddressable)] != 1 {
		t.Fatalf(
			"observed offers = %+v, want 1 posting for outcome %q",
			observer.postingsOffered, postingOfferUnaddressable,
		)
	}
	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none for an unaddressable peer", replicas)
	}
}

func TestPostingOfferCycleExcludesPostingWhenURLMetadataDeliveryFails(t *testing.T) {
	now := time.Unix(1000, 0)
	word, url := yacymodel.WordHash("w1"), urlHash("u1")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, url): fakePosting(word, url),
	}
	schedule, ledger, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{
		Outcome:           postingOfferAccepted,
		URLsUnknownToPeer: []yacymodel.URLHash{url},
	}
	cycle.delivery.urls = fakeURLDirectory{
		metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
			url: {Address: "http://example.com/u1"},
		},
	}
	cycle.delivery.urlMetadataCourier = &fakeURLMetadataCourier{
		receipt: urlMetadataReceipt{Outcome: urlMetadataDeferred},
	}

	store(t, schedule, word, url)

	cycle.runCycle(context.Background())

	replicas, err := ledger.Replicas(context.Background(), word, url)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none recorded when url metadata delivery fails", replicas)
	}
	if observer.urlMetadataDeliveries[string(urlMetadataDeferred)] != 1 {
		t.Fatalf(
			"observed url metadata deliveries = %+v, want 1 for outcome %q",
			observer.urlMetadataDeliveries, urlMetadataDeferred,
		)
	}

	future := now.Add(cycle.settlement.cadence.retry + time.Second)
	schedule.now = func() time.Time { return future }
	due, err := schedule.DuePostings(context.Background(), 10)
	if err != nil {
		t.Fatalf("DuePostings: %v", err)
	}
	if len(due) != 1 || due[0].Word != word {
		t.Fatalf(
			"due = %v, want [word] retried after a failed url metadata delivery",
			due,
		)
	}
}

func TestPostingOfferCycleDeliversMetadataItHasWhenOneURLIsAbsent(t *testing.T) {
	now := time.Unix(1000, 0)
	word := yacymodel.WordHash("w1")
	present, absent := urlHash("u1"), urlHash("u2")
	peer := yacymodel.WordHash("peer")
	postings := map[yacymodel.Hash]yacymodel.RWIPosting{
		fakePostingKey(word, present): fakePosting(word, present),
		fakePostingKey(word, absent):  fakePosting(word, absent),
	}
	schedule, ledger, courier, observer, cycle := openPostingOfferCycle(
		t, func() time.Time { return now }, postings, []yacymodel.Seed{seed(peer)},
	)
	courier.receipts[peer] = postingOfferReceipt{
		Outcome:           postingOfferAccepted,
		URLsUnknownToPeer: []yacymodel.URLHash{present, absent},
	}
	cycle.delivery.urls = fakeURLDirectory{
		metadata: map[yacymodel.URLHash]yacymodel.URLMetadata{
			present: {Address: "http://example.com/u1"},
		},
	}
	metadataCourier := &fakeURLMetadataCourier{
		receipt: urlMetadataReceipt{Outcome: urlMetadataAccepted},
	}
	cycle.delivery.urlMetadataCourier = metadataCourier

	store(t, schedule, word, present)
	store(t, schedule, word, absent)

	cycle.runCycle(context.Background())

	if len(metadataCourier.delivered) != 1 {
		t.Fatalf(
			"delivered = %v, want the one url whose metadata this node holds",
			metadataCourier.delivered,
		)
	}
	if observer.urlMetadataDeliveries[string(urlMetadataUnavailable)] != 1 {
		t.Fatalf(
			"observed url metadata deliveries = %+v, want 1 for outcome %q",
			observer.urlMetadataDeliveries, urlMetadataUnavailable,
		)
	}

	replicas, err := ledger.Replicas(context.Background(), word, present)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 1 {
		t.Fatalf("replicas = %v, want the deliverable posting recorded", replicas)
	}

	replicas, err = ledger.Replicas(context.Background(), word, absent)
	if err != nil {
		t.Fatalf("Replicas: %v", err)
	}
	if len(replicas) != 0 {
		t.Fatalf("replicas = %v, want none for the posting with no url metadata", replicas)
	}
}
