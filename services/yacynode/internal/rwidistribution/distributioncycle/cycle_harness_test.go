package distributioncycle

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicashortfall"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/urlmetadatacourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type cycleOptions struct {
	postings        map[yacymodel.Hash]yacymodel.RWIPosting
	postingsErr     error
	roster          fakeRoster
	reachability    replicashortfall.Reachability
	redundancy      int
	urls            fakeURLDirectory
	metadataOutcome urlmetadatacourier.Outcome
}

type cycleHarness struct {
	v               *vault.Vault
	clk             *clock
	schedule        *postingschedule.Schedule
	replicas        *postingreplicas.Replicas
	shortfall       *replicashortfall.Shortfall
	courier         *fakeCourier
	metadataCourier *fakeURLMetadataCourier
	urls            fakeURLDirectory
	observer        *fakeObserver
	delivery        *OfferDelivery
	cadence         Cadence
	cycle           *Cycle
}

func openCycle(t *testing.T, clk *clock, opts cycleOptions) *cycleHarness {
	t.Helper()

	if opts.reachability == nil {
		opts.reachability = opts.roster
	}
	redundancy := opts.redundancy
	if redundancy == 0 {
		redundancy = 1
	}
	if opts.metadataOutcome == "" {
		opts.metadataOutcome = urlmetadatacourier.Accepted
	}

	v, schedule, replicas := openCycleVault(t, clk.now)

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	shortfall := replicashortfall.New(
		schedule,
		replicas,
		fakePostingIndex{postings: opts.postings, unread: opts.postingsErr},
		opts.reachability,
		partitions,
		redundancy,
	)

	courier := &fakeCourier{receipts: make(map[yacymodel.Hash]postingcourier.PostingReceipt)}
	metadataCourier := &fakeURLMetadataCourier{
		receipt: urlmetadatacourier.URLMetadataReceipt{Outcome: opts.metadataOutcome},
	}
	urls := opts.urls
	observer := newFakeObserver()

	delivery := NewOfferDelivery(courier, metadataCourier, urls, observer)
	cadence := Cadence{Refresh: time.Hour, Backoff: time.Minute}
	cycle := New(
		shortfall,
		delivery,
		replicas,
		cadence,
		schedule,
		opts.roster,
		observer,
		clk.now,
		10,
		time.Minute,
		0,
	)

	return &cycleHarness{
		v:               v,
		clk:             clk,
		schedule:        schedule,
		replicas:        replicas,
		shortfall:       shortfall,
		courier:         courier,
		metadataCourier: metadataCourier,
		urls:            urls,
		observer:        observer,
		delivery:        delivery,
		cadence:         cadence,
		cycle:           cycle,
	}
}

func openCycleVault(
	t *testing.T,
	now func() time.Time,
) (*vault.Vault, *postingschedule.Schedule, *postingreplicas.Replicas) {
	t.Helper()

	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	schedule, err := postingschedule.Open(v, now)
	if err != nil {
		t.Fatalf("postingschedule.Open: %v", err)
	}
	replicas, err := postingreplicas.Open(v, schedule)
	if err != nil {
		t.Fatalf("postingreplicas.Open: %v", err)
	}

	return v, schedule, replicas
}

type fakePostingIndex struct {
	postings map[yacymodel.Hash]yacymodel.RWIPosting
	unread   error
}

func (f fakePostingIndex) RWICount(context.Context) (int, error) { return len(f.postings), nil }

func (f fakePostingIndex) Posting(
	_ context.Context,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	if f.unread != nil {
		return yacymodel.RWIPosting{}, false, f.unread
	}
	entry, found := f.postings[fakePostingKey(word, url)]

	return entry, found, nil
}

func (f fakePostingIndex) ScanWord(
	context.Context,
	yacymodel.Hash,
	func(yacymodel.RWIPosting) (bool, error),
) error {
	return nil
}

type fakeRoster struct {
	reachable         []yacymodel.Seed
	recentlyReachable map[yacymodel.Hash]struct{}
}

func (fakeRoster) Discover(context.Context, ...yacymodel.Seed)            {}
func (fakeRoster) ConfirmReachable(context.Context, yacymodel.Hash)       {}
func (fakeRoster) ConfirmUnreachable(context.Context, yacymodel.Hash)     {}
func (fakeRoster) UnreachablePeers(context.Context, int) []yacymodel.Seed { return nil }

func (f fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	return f.reachable
}

func (f fakeRoster) Reachable(_ context.Context, peer yacymodel.Hash) bool {
	for _, seed := range f.reachable {
		if seed.Hash == peer {
			return true
		}
	}

	return false
}

func (f fakeRoster) RecentlyReachable(_ context.Context, peer yacymodel.Hash) bool {
	_, recent := f.recentlyReachable[peer]

	return recent
}

type fakeCourier struct {
	receipts map[yacymodel.Hash]postingcourier.PostingReceipt
	offered  []offer
	onOffer  func()
}

func (f *fakeCourier) Offer(
	_ context.Context,
	_ string,
	recipient yacymodel.Seed,
	postings []yacymodel.RWIPosting,
) postingcourier.PostingReceipt {
	f.offered = append(f.offered, offer{Peer: recipient, Postings: postings})
	if f.onOffer != nil {
		f.onOffer()
	}

	return f.receipts[recipient.Hash]
}

type fakeURLMetadataCourier struct {
	receipt   urlmetadatacourier.URLMetadataReceipt
	delivered []yacymodel.URLMetadata
}

func (f *fakeURLMetadataCourier) Deliver(
	_ context.Context,
	_ string,
	_ yacymodel.Hash,
	metadata []yacymodel.URLMetadata,
) urlmetadatacourier.URLMetadataReceipt {
	f.delivered = append(f.delivered, metadata...)

	return f.receipt
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

func store(
	t *testing.T,
	v *vault.Vault,
	schedule *postingschedule.Schedule,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return schedule.PostingStored(tx, word, url)
	}); err != nil {
		t.Fatalf("PostingStored: %v", err)
	}
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
