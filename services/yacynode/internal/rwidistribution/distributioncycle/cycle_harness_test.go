package distributioncycle_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferinterval"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
)

const (
	cycleEndWait     = 10 * time.Second
	postingsPerBatch = 10
	drainBudget      = time.Minute
)

type cycleOptions struct {
	postings          map[yacymodel.Hash]yacymodel.RWIPosting
	postingsErr       error
	roster            fakeRoster
	reachability      postingoffer.Reachability
	redundancy        int
	self              yacymodel.Hash
	cooldown          time.Duration
	minReachablePeers int
	drainBudget       time.Duration
	batchConcurrency  int
}

type cycleHarness struct {
	v             *vault.Vault
	clk           *clock
	postings      *fakePostingIndex
	schedule      *postingofferschedule.Schedule
	replicas      *postingreplicas.Replicas
	offers        *postingoffer.PostingOffers
	eligibility   *replicaeligibility.Peers
	observer      *fakeObserver
	transfers     *fakeTransfers
	offerInterval postingofferinterval.Bounds
	cycle         *distributioncycle.Cycle
}

func withCycleDefaults(opts cycleOptions) cycleOptions {
	if opts.reachability == nil {
		opts.reachability = opts.roster
	}
	if opts.redundancy == 0 {
		opts.redundancy = 1
	}
	if opts.self == (yacymodel.Hash{}) {
		opts.self = thisNodeFartherThanEveryPeer()
	}
	if opts.drainBudget == 0 {
		opts.drainBudget = drainBudget
	}
	if opts.batchConcurrency == 0 {
		opts.batchConcurrency = 1
	}

	return opts
}

func openCycle(t *testing.T, clk *clock, opts cycleOptions) *cycleHarness {
	t.Helper()

	opts = withCycleDefaults(opts)

	observer := newFakeObserver()
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	offerInterval := postingofferinterval.Bounds{Shortest: time.Minute, Longest: time.Hour}
	v, schedule, replicas := openCycleVault(t, partitions, offerInterval, clk.now, observer)

	eligibility := replicaeligibility.New(opts.cooldown, clk.now)
	postings := &fakePostingIndex{postings: opts.postings, unread: opts.postingsErr}
	postings.purged = purgeBookkeeping(schedule, replicas)
	offers := postingoffer.New(
		v,
		schedule,
		replicas,
		postings,
		opts.reachability,
		eligibility,
		observer,
		postingoffer.Config{
			Partitions: partitions,
			Self:       opts.self,
			Redundancy: opts.redundancy,
		},
	)
	handoff := postinghandoff.New(
		postings,
		postinghandoff.Config{
			Self:       opts.self,
			Redundancy: opts.redundancy,
		},
	)

	transfers := &fakeTransfers{answers: make(map[yacymodel.Hash]fakeAnswer)}
	cycle := distributioncycle.New(
		v,
		offers,
		handoff,
		transfers,
		eligibility,
		replicas,
		schedule,
		opts.roster,
		clk.now,
		observer,
		observer,
		distributioncycle.Config{
			PostingsPerBatch:  postingsPerBatch,
			BatchConcurrency:  opts.batchConcurrency,
			CycleInterval:     time.Minute,
			DrainBudget:       opts.drainBudget,
			MinReachablePeers: opts.minReachablePeers,
		},
	)

	return &cycleHarness{
		v:             v,
		clk:           clk,
		postings:      postings,
		schedule:      schedule,
		replicas:      replicas,
		offers:        offers,
		eligibility:   eligibility,
		observer:      observer,
		transfers:     transfers,
		offerInterval: offerInterval,
		cycle:         cycle,
	}
}

func (h *cycleHarness) duePostings(t *testing.T, limit int) []postingidentity.Identity {
	t.Helper()

	var due []postingidentity.Identity
	if err := h.v.View(context.Background(), func(tx *vault.Txn) error {
		for sector := range yacymodel.DHTRingSectorCount {
			inSector, err := h.schedule.DuePostingsIn(tx, sector, limit-len(due))
			if err != nil {
				return err
			}
			due = append(due, inSector...)
		}

		return nil
	}); err != nil {
		t.Fatalf("DuePostings: %v", err)
	}

	return due
}

func (h *cycleHarness) runCycle(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		h.cycle.Run(ctx)
	}()

	select {
	case <-h.observer.cycleEnds:
	case <-time.After(cycleEndWait):
		cancel()
		t.Fatal("timed out waiting for the distribution cycle to end")
	}

	cancel()
	<-stopped
}

func purgeBookkeeping(
	schedule *postingofferschedule.Schedule,
	replicas *postingreplicas.Replicas,
) func(*vault.Txn, yacymodel.Hash, yacymodel.URLHash) error {
	return func(tx *vault.Txn, word yacymodel.Hash, url yacymodel.URLHash) error {
		purgedPosting := yacymodel.RWIPosting{WordHash: word, URLHash: url}
		if err := schedule.PostingPurged(tx, purgedPosting); err != nil {
			return err
		}

		return replicas.PostingPurged(tx, purgedPosting)
	}
}

func openCycleVault(
	t *testing.T,
	partitions yacymodel.DHTRingPartitions,
	offerInterval postingofferinterval.Bounds,
	now func() time.Time,
	observer postingofferschedule.Observer,
) (
	*vault.Vault,
	*postingofferschedule.Schedule,
	*postingreplicas.Replicas,
) {
	t.Helper()

	v, err := vault.New(
		vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(0)),
		nil,
	)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := postingofferschedule.Open(v, partitions, offerInterval, now, observer)
	if err != nil {
		t.Fatalf("postingofferschedule.Open: %v", err)
	}
	replicas, err := postingreplicas.Open(v)
	if err != nil {
		t.Fatalf("postingreplicas.Open: %v", err)
	}

	return v, schedule, replicas
}

type fakePostingIndex struct {
	guard    sync.Mutex
	postings map[yacymodel.Hash]yacymodel.RWIPosting
	unread   error
	purged   func(tx *vault.Txn, word yacymodel.Hash, url yacymodel.URLHash) error
	onRead   func()
}

func (f *fakePostingIndex) PurgePosting(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (bool, error) {
	f.guard.Lock()
	defer f.guard.Unlock()

	key := fakePostingKey(word, url)
	if _, found := f.postings[key]; !found {
		return false, nil
	}
	tx.RunAfterCommit(func() {
		f.guard.Lock()
		defer f.guard.Unlock()

		delete(f.postings, key)
	})

	return true, f.purged(tx, word, url)
}

func (f *fakePostingIndex) PostingOf(
	_ *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	if f.onRead != nil {
		f.onRead()
	}
	f.guard.Lock()
	defer f.guard.Unlock()

	if f.unread != nil {
		return yacymodel.RWIPosting{}, false, f.unread
	}
	entry, found := f.postings[fakePostingKey(word, url)]

	return entry, found, nil
}

type fakeRoster struct {
	reachable         []yacymodel.Seed
	recentlyReachable map[yacymodel.Hash]struct{}
}

func (f fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	return f.reachable
}

func (f fakeRoster) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	for _, seed := range f.reachable {
		if seed.Hash == peer {
			return true
		}
	}

	return false
}

func (f fakeRoster) IsRecentlyReachable(_ context.Context, peer yacymodel.Hash) bool {
	_, recent := f.recentlyReachable[peer]

	return recent
}

type offeredCall struct {
	Peer     yacymodel.Seed
	Postings []yacymodel.RWIPosting
}

type fakeAnswer struct {
	accepted       bool
	requestedPause time.Duration
}

var (
	acceptedAnswer = fakeAnswer{accepted: true}
	declinedAnswer = fakeAnswer{}
)

type fakeTransfers struct {
	guard   sync.Mutex
	answers map[yacymodel.Hash]fakeAnswer
	offered []offeredCall
	onOffer func()
}

func (f *fakeTransfers) Send(
	_ context.Context,
	peer yacymodel.Seed,
	offeredPostings []yacymodel.RWIPosting,
) postingtransfer.OfferAnswer {
	f.guard.Lock()
	f.offered = append(f.offered, offeredCall{Peer: peer, Postings: offeredPostings})
	answer := f.answers[peer.Hash]
	f.guard.Unlock()
	if f.onOffer != nil {
		f.onOffer()
	}
	if !answer.accepted {
		return postingtransfer.OfferAnswer{RequestedPause: answer.requestedPause}
	}

	return postingtransfer.OfferAnswer{
		Accepted:         true,
		AcceptedPostings: offeredPostings,
		RequestedPause:   answer.requestedPause,
	}
}

func store(
	t *testing.T,
	v *vault.Vault,
	schedule *postingofferschedule.Schedule,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingStored(tx, yacymodel.RWIPosting{WordHash: word, URLHash: url})
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
}

func recordAccepted(
	t *testing.T,
	h *cycleHarness,
	peer yacymodel.Hash,
	postings ...yacymodel.RWIPosting,
) {
	t.Helper()

	if err := h.v.Update(context.Background(), func(tx *vault.Txn) error {
		for _, posting := range postings {
			identity := postingidentity.IdentityOf(posting)
			recordedHolders, err := h.replicas.HoldersOf(tx, identity)
			if err != nil {
				return err
			}
			change := postingreplicas.HolderChange{AcceptingPeers: []yacymodel.Hash{peer}}
			if err := h.replicas.RecordHolders(
				tx, identity, recordedHolders, change.HoldersFrom(recordedHolders),
			); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("RecordHolders: %v", err)
	}
}

func holdersOf(
	t *testing.T,
	h *cycleHarness,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) []yacymodel.Hash {
	t.Helper()

	var holders []yacymodel.Hash
	if err := h.v.View(context.Background(), func(tx *vault.Txn) error {
		var err error
		holders, err = h.replicas.HoldersOf(tx, postingidentity.Identity{Word: word, URL: url})

		return err
	}); err != nil {
		t.Fatalf("Holders: %v", err)
	}

	return holders
}

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

func seed(hash yacymodel.Hash) yacymodel.Seed {
	host, err := yacymodel.ParseHost("192.0.2.1")
	if err != nil {
		panic(err)
	}
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		panic(err)
	}

	return yacymodel.Seed{
		Hash:           hash,
		PrimaryAddress: yacymodel.Some(host),
		Port:           yacymodel.Some(port),
		Capabilities:   yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
	}
}

func thisNodeFartherThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self5") }

func thisNodeCloserThanEveryPeer() yacymodel.Hash { return yacymodel.WordHash("self22") }

func thisNodeFartherThanTheClosestPeer() yacymodel.Hash { return yacymodel.WordHash("self15") }

func fakePosting(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: word, URLHash: url}
}

func fakePostingKey(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.Hash {
	return yacymodel.WordHash(word.String() + url.String())
}

func urlHash(raw string) yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash(raw).String())
	if err != nil {
		panic(err)
	}

	return hash
}
