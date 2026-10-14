package distributioncycle_test

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/distributioncycle"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicaeligibility"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionhandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const (
	cycleEndWait     = 10 * time.Second
	postingsPerBatch = 10
	drainBudget      = time.Minute
	refreshInterval  = time.Hour
	retryDelay       = 10 * time.Minute
	positionStep     = 8
)

type cycleOptions struct {
	self              yacymodel.Hash
	redundancy        int
	cooldown          time.Duration
	minReachablePeers int
	drainBudget       time.Duration
	batchConcurrency  int
	postingsPerBatch  int
	handoffEnabled    bool
}

type cycleHarness struct {
	v           *vault.Vault
	clk         *clock
	index       *observedPostingIndex
	admitter    rwipostings.PostingAdmitter
	undelivered *undeliveredpostings.UndeliveredPostings
	holders     *wordpartitionholders.Holders
	roster      *fakeRoster
	observer    *fakeObserver
	transfers   *fakeTransfers
	cycle       *distributioncycle.Cycle
}

func withCycleDefaults(opts cycleOptions) cycleOptions {
	if opts.redundancy == 0 {
		opts.redundancy = 1
	}
	if opts.self.IsZero() {
		opts.self = thisNodeFartherThan(fixtureWordPartition())
	}
	if opts.drainBudget == 0 {
		opts.drainBudget = drainBudget
	}
	if opts.batchConcurrency == 0 {
		opts.batchConcurrency = 1
	}
	if opts.postingsPerBatch == 0 {
		opts.postingsPerBatch = postingsPerBatch
	}

	return opts
}

func openCycle(t *testing.T, clk *clock, opts cycleOptions) *cycleHarness {
	t.Helper()

	opts = withCycleDefaults(opts)
	v := openCycleVault(t)
	undelivered, projection, err := undeliveredpostings.Open(v, partitions())
	if err != nil {
		t.Fatalf("undeliveredpostings.Open: %v", err)
	}
	index, admitter, purger, err := rwipostings.Open(
		context.Background(),
		v,
		partitions(),
		[]vault.Projection[yacymodel.RWIPosting]{projection},
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	holders, err := wordpartitionholders.Open(v, partitions(), refreshInterval, retryDelay, clk.now)
	if err != nil {
		t.Fatalf("wordpartitionholders.Open: %v", err)
	}

	h := &cycleHarness{
		v:           v,
		clk:         clk,
		index:       &observedPostingIndex{index: index},
		admitter:    admitter,
		undelivered: undelivered,
		holders:     holders,
		roster:      &fakeRoster{recentlyReachable: map[yacymodel.Hash]struct{}{}},
		observer:    newFakeObserver(),
		transfers:   &fakeTransfers{answers: make(map[yacymodel.Hash]fakeAnswer)},
	}
	cooldowns := replicaeligibility.New(opts.cooldown, clk.now)
	h.cycle = distributioncycle.New(
		v,
		wordpartitionoffer.New(
			v, undelivered, holders, h.index, h.roster, cooldowns, h.observer,
			wordpartitionoffer.Config{
				Partitions:       partitions(),
				Self:             opts.self,
				Redundancy:       opts.redundancy,
				PostingsPerBatch: opts.postingsPerBatch,
			},
		),
		undelivered,
		holders,
		handoffFor(opts, h.index, purger, undelivered, holders),
		h.transfers,
		cooldowns,
		h.roster,
		clk.now,
		h.observer,
		h.observer,
		distributioncycle.Config{
			PostingsPerBatch:  opts.postingsPerBatch,
			BatchConcurrency:  opts.batchConcurrency,
			CycleInterval:     time.Minute,
			DrainBudget:       opts.drainBudget,
			MinReachablePeers: opts.minReachablePeers,
		},
	)

	return h
}

func openCycleVault(t *testing.T) *vault.Vault {
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

	return v
}

func handoffFor(
	opts cycleOptions,
	index *observedPostingIndex,
	purger rwipostings.PostingPurger,
	undelivered *undeliveredpostings.UndeliveredPostings,
	holders *wordpartitionholders.Holders,
) distributioncycle.WordPartitionHandoff {
	if !opts.handoffEnabled {
		return wordpartitionhandoff.KeepEveryWordPartition{}
	}

	return wordpartitionhandoff.New(index, purger, undelivered, holders,
		wordpartitionhandoff.Config{
			Partitions: partitions(),
			Self:       opts.self,
			Redundancy: opts.redundancy,
		})
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

func (h *cycleHarness) reach(peers ...yacymodel.Seed) {
	h.roster.reachable = peers
}

func (h *cycleHarness) admit(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	urls ...yacymodel.URLHash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, url := range urls {
			if err := h.admitter.Admit(tx, postingOf(wordPartition, url)); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *cycleHarness) recordHolders(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	holders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.RecordAtRedundancy(tx, wordPartition, holders)
	})
}

func (h *cycleHarness) holdersOf(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
) []yacymodel.Hash {
	t.Helper()

	var recordedHolders []yacymodel.Hash
	h.read(t, func(tx *vault.Txn) error {
		holders, err := h.holders.Of(tx, wordPartition)
		recordedHolders = holders

		return err
	})

	return recordedHolders
}

func (h *cycleHarness) amountOfUndeliveredPostings(t *testing.T) int {
	t.Helper()

	var amountOfPostings int
	h.read(t, func(tx *vault.Txn) error {
		amount, err := h.undelivered.AmountOfPostings(tx)
		amountOfPostings = amount

		return err
	})

	return amountOfPostings
}

func (h *cycleHarness) postingsOf(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
) []yacymodel.URLHash {
	t.Helper()

	var urls []yacymodel.URLHash
	h.read(t, func(tx *vault.Txn) error {
		return h.index.index.PostingsOfWordIn(
			tx,
			wordPartition.Word,
			wordPartition.Partition,
			func(posting yacymodel.RWIPosting) (bool, error) {
				urls = append(urls, posting.URLHash)

				return true, nil
			},
		)
	})

	return urls
}

func (h *cycleHarness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.v.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h *cycleHarness) read(t *testing.T, view func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.v.View(context.Background(), view); err != nil {
		t.Fatalf("View: %v", err)
	}
}

type observedPostingIndex struct {
	guard  sync.Mutex
	index  rwipostings.PostingIndex
	unread error
	onRead func()
}

func (o *observedPostingIndex) PostingOf(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	if err := o.read(); err != nil {
		return yacymodel.RWIPosting{}, false, err
	}

	return o.index.PostingOf(tx, word, url)
}

func (o *observedPostingIndex) PostingsOfWordIn(
	tx *vault.Txn,
	word yacymodel.Hash,
	partition uint,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	if err := o.read(); err != nil {
		return err
	}

	return o.index.PostingsOfWordIn(tx, word, partition, visit)
}

func (o *observedPostingIndex) read() error {
	if o.onRead != nil {
		o.onRead()
	}
	o.guard.Lock()
	defer o.guard.Unlock()

	return o.unread
}

type fakeRoster struct {
	reachable         []yacymodel.Seed
	recentlyReachable map[yacymodel.Hash]struct{}
}

func (f *fakeRoster) ReachablePeers(context.Context) []yacymodel.Seed {
	return f.reachable
}

func (f *fakeRoster) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	return slices.ContainsFunc(f.reachable, func(seed yacymodel.Seed) bool {
		return seed.Hash == peer
	})
}

func (f *fakeRoster) IsRecentlyReachable(_ context.Context, peer yacymodel.Hash) bool {
	_, recent := f.recentlyReachable[peer]

	return recent
}

type offeredCall struct {
	Peer     yacymodel.Seed
	Postings []yacymodel.RWIPosting
}

type fakeAnswer struct {
	accepted         bool
	acceptedPostings int
	requestedPause   time.Duration
}

var (
	acceptedAnswer = fakeAnswer{accepted: true}
	declinedAnswer = fakeAnswer{}
)

func acceptedInPart(acceptedPostings int) fakeAnswer {
	return fakeAnswer{accepted: true, acceptedPostings: acceptedPostings}
}

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
	acceptedPostings := offeredPostings
	if answer.acceptedPostings > 0 {
		acceptedPostings = offeredPostings[:min(answer.acceptedPostings, len(offeredPostings))]
	}

	return postingtransfer.OfferAnswer{
		Accepted:         true,
		AcceptedPostings: acceptedPostings,
		RequestedPause:   answer.requestedPause,
	}
}

func (f *fakeTransfers) offeredTo(peer yacymodel.Seed) []offeredCall {
	f.guard.Lock()
	defer f.guard.Unlock()

	var offeredToPeer []offeredCall
	for _, call := range f.offered {
		if call.Peer.Hash == peer.Hash {
			offeredToPeer = append(offeredToPeer, call)
		}
	}

	return offeredToPeer
}

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

func startClock() *clock { return &clock{at: time.Unix(1_700_000_000, 0).UTC()} }

func partitions() yacymodel.DHTRingPartitions {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(0)
	if err != nil {
		panic(err)
	}

	return partitions
}

func wordPartitionOf(word int) yacymodel.WordPartition {
	return yacymodel.WordPartition{Word: yacymodel.WordHash("word" + strconv.Itoa(word))}
}

func fixtureWordPartition() yacymodel.WordPartition { return wordPartitionOf(0) }

func positionOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOfWordPartition(wordPartition, partitions())
}

func sectorOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(positionOf(wordPartition))
}

func wordPartitionsIn(
	sector yacymodel.DHTRingSector,
	amount int,
) []yacymodel.WordPartition {
	var wordPartitions []yacymodel.WordPartition
	for word := 0; len(wordPartitions) < amount; word++ {
		if wordPartition := wordPartitionOf(word); sectorOf(wordPartition) == sector {
			wordPartitions = append(wordPartitions, wordPartition)
		}
	}
	slices.SortFunc(wordPartitions, func(wordPartition, other yacymodel.WordPartition) int {
		return cmp.Compare(positionOf(wordPartition), positionOf(other))
	})

	return wordPartitions
}

func urlsOf(amount int) []yacymodel.URLHash {
	urls := make([]yacymodel.URLHash, 0, amount)
	for document := range amount {
		url, err := yacymodel.ParseURLHash(
			yacymodel.WordHash("url" + strconv.Itoa(document)).String(),
		)
		if err != nil {
			panic(err)
		}
		urls = append(urls, url)
	}

	return urls
}

func postingOf(wordPartition yacymodel.WordPartition, url yacymodel.URLHash) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash: wordPartition.Word,
		URLHash:  url,
		Language: yacymodel.LanguageOfUndeclaredDocument,
	}
}

func peerAfter(
	wordPartition yacymodel.WordPartition,
	steps yacymodel.DHTRingPosition,
) yacymodel.Seed {
	return seed(yacymodel.HashFromDHTRingPosition(positionOf(wordPartition) + steps*positionStep))
}

func thisNodeFartherThan(wordPartition yacymodel.WordPartition) yacymodel.Hash {
	return yacymodel.HashFromDHTRingPosition(positionOf(wordPartition) - positionStep)
}

func thisNodeCloserThanEveryPeerOf(wordPartition yacymodel.WordPartition) yacymodel.Hash {
	return yacymodel.HashFromDHTRingPosition(positionOf(wordPartition))
}

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

func urlsIn(calls []offeredCall) []yacymodel.URLHash {
	var urls []yacymodel.URLHash
	for _, call := range calls {
		for _, posting := range call.Postings {
			urls = append(urls, posting.URLHash)
		}
	}

	return sortedURLs(urls)
}

func sortedURLs(urls []yacymodel.URLHash) []yacymodel.URLHash {
	return slices.SortedFunc(slices.Values(urls), func(url, other yacymodel.URLHash) int {
		return cmp.Compare(url.String(), other.String())
	})
}
