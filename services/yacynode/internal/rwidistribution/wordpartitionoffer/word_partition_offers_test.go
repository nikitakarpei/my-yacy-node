package wordpartitionoffer_test

import (
	"cmp"
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const (
	partitionExponent = 2
	refreshInterval   = time.Hour
	retryDelay        = time.Minute
	positionStep      = 8
	defaultBatchSize  = 100
)

type options struct {
	self             yacymodel.Hash
	redundancy       int
	postingsPerBatch int
}

type harness struct {
	vault        *vault.Vault
	admitter     rwipostings.PostingAdmitter
	undelivered  *undeliveredpostings.UndeliveredPostings
	holders      *wordpartitionholders.Holders
	offers       *wordpartitionoffer.WordPartitionOffers
	reachability *fakeReachability
	eligibility  fakeEligibility
	observer     *fakeObserver
	clock        time.Time
}

func openHarness(t *testing.T, opts options) *harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	undelivered, projection, err := undeliveredpostings.Open(v, partitions())
	if err != nil {
		t.Fatalf("undeliveredpostings.Open: %v", err)
	}
	index, admitter, _, err := rwipostings.Open(
		context.Background(),
		v,
		partitions(),
		[]vault.Projection[yacymodel.RWIPosting]{projection},
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	h := &harness{
		vault:        v,
		admitter:     admitter,
		undelivered:  undelivered,
		reachability: &fakeReachability{recentlyReachable: map[yacymodel.Hash]bool{}},
		eligibility: fakeEligibility{
			cooldowns:     map[yacymodel.Hash]time.Duration{},
			excludedPeers: map[yacymodel.Hash]bool{},
		},
		observer: &fakeObserver{},
		clock:    time.Unix(1_700_000_000, 0).UTC(),
	}
	h.holders, err = wordpartitionholders.Open(v, partitions(), refreshInterval, retryDelay, h.now)
	if err != nil {
		t.Fatalf("wordpartitionholders.Open: %v", err)
	}
	h.offers = wordpartitionoffer.New(
		v, undelivered, h.holders, index, h.reachability, h.eligibility, h.observer,
		configFrom(opts),
	)

	return h
}

func configFrom(opts options) wordpartitionoffer.Config {
	config := wordpartitionoffer.Config{
		Partitions:       partitions(),
		Self:             opts.self,
		Redundancy:       opts.redundancy,
		PostingsPerBatch: opts.postingsPerBatch,
	}
	if config.Self.IsZero() {
		config.Self = yacymodel.HashFromDHTRingPosition(
			yacymodel.LowestDHTRingPositionIn(fixtureSector()),
		)
	}
	if config.Redundancy == 0 {
		config.Redundancy = 1
	}
	if config.PostingsPerBatch == 0 {
		config.PostingsPerBatch = defaultBatchSize
	}

	return config
}

func (h *harness) now() time.Time {
	return h.clock
}

func (h *harness) admit(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	urls ...yacymodel.URLHash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, url := range urls {
			posting := yacymodel.RWIPosting{
				WordHash: wordPartition.Word,
				URLHash:  url,
				Language: yacymodel.LanguageOfUndeclaredDocument,
			}
			if err := h.admitter.Admit(tx, posting); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *harness) deliverEverything(t *testing.T) {
	t.Helper()

	var undelivered []undeliveredpostings.UndeliveredPosting
	h.read(t, func(tx *vault.Txn) error {
		return h.undelivered.PostingsIn(
			tx,
			fixtureSector(),
			undeliveredpostings.UndeliveredPosting{},
			func(posting undeliveredpostings.UndeliveredPosting) (bool, error) {
				undelivered = append(undelivered, posting)

				return true, nil
			},
		)
	})
	h.write(t, func(tx *vault.Txn) error {
		for _, posting := range undelivered {
			if _, err := h.undelivered.MarkDelivered(tx, posting); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *harness) recordHolders(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	holders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.RecordAtRedundancy(tx, wordPartition, holders)
	})
}

func (h *harness) recordNoHolders(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.RecordShortOfRedundancy(tx, wordPartition, nil)
	})
}

func (h *harness) dueNow(
	t *testing.T,
	from wordpartitionoffer.Cursor,
	reachablePeers ...yacymodel.Seed,
) ([]wordpartitionoffer.WordPartitionOffer, wordpartitionoffer.Cursor) {
	t.Helper()

	batch := h.dueBatch(t, from, reachablePeers...)

	return batch.Offers, batch.Next
}

func (h *harness) dueBatch(
	t *testing.T,
	from wordpartitionoffer.Cursor,
	reachablePeers ...yacymodel.Seed,
) wordpartitionoffer.DueBatch {
	t.Helper()

	h.reachability.reachable = reachablePeers
	batch, err := h.offers.DueNowIn(
		context.Background(),
		fixtureSector(),
		from,
		reachablePeers,
	)
	if err != nil {
		t.Fatalf("DueNowIn: %v", err)
	}

	return batch
}

func (h *harness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h *harness) read(t *testing.T, view func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.View(context.Background(), view); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func partitions() yacymodel.DHTRingPartitions {
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		panic(err)
	}

	return partitions
}

func fixturePartition() uint {
	return partitions().PartitionOf(urlHashOf(0))
}

func fixtureSector() yacymodel.DHTRingSector {
	return sectorOf(yacymodel.WordPartition{Word: wordOf(0), Partition: fixturePartition()})
}

func sectorOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(positionOf(wordPartition))
}

func positionOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOfWordPartition(wordPartition, partitions())
}

func wordPartitionsInRingOrder(t *testing.T, amount int) []yacymodel.WordPartition {
	t.Helper()

	var wordPartitions []yacymodel.WordPartition
	for word := 0; len(wordPartitions) < amount; word++ {
		wordPartition := yacymodel.WordPartition{Word: wordOf(word), Partition: fixturePartition()}
		if sectorOf(wordPartition) == fixtureSector() {
			wordPartitions = append(wordPartitions, wordPartition)
		}
	}
	slices.SortFunc(wordPartitions, func(wordPartition, other yacymodel.WordPartition) int {
		return cmp.Compare(positionOf(wordPartition), positionOf(other))
	})

	return wordPartitions
}

func urlsOfPartition(amount int) []yacymodel.URLHash {
	var urls []yacymodel.URLHash
	for document := 0; len(urls) < amount; document++ {
		if url := urlHashOf(document); partitions().PartitionOf(url) == fixturePartition() {
			urls = append(urls, url)
		}
	}

	return urls
}

func wordOf(word int) yacymodel.Hash {
	return yacymodel.WordHash("word" + strconv.Itoa(word))
}

func urlHashOf(document int) yacymodel.URLHash {
	url, err := yacymodel.ParseURLHash(yacymodel.WordHash("url" + strconv.Itoa(document)).String())
	if err != nil {
		panic(err)
	}

	return url
}

func peerAfter(
	wordPartition yacymodel.WordPartition,
	steps yacymodel.DHTRingPosition,
) yacymodel.Seed {
	return seedAt(positionOf(wordPartition) + steps*positionStep)
}

func seedAt(position yacymodel.DHTRingPosition) yacymodel.Seed {
	host, err := yacymodel.ParseHost("192.0.2.1")
	if err != nil {
		panic(err)
	}
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		panic(err)
	}

	return yacymodel.Seed{
		Hash:           yacymodel.HashFromDHTRingPosition(position),
		PrimaryAddress: yacymodel.Some(host),
		Port:           yacymodel.Some(port),
		Capabilities:   yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
	}
}

func onlyOffer(
	t *testing.T,
	offers []wordpartitionoffer.WordPartitionOffer,
) wordpartitionoffer.WordPartitionOffer {
	t.Helper()

	if len(offers) != 1 {
		t.Fatalf("offers = %+v, want one", offers)
	}

	return offers[0]
}

func wordPartitionsOf(offers []wordpartitionoffer.WordPartitionOffer) []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(offers))
	for _, offer := range offers {
		wordPartitions = append(wordPartitions, offer.WordPartition)
	}

	return wordPartitions
}

func recipientPeersOf(offer wordpartitionoffer.WordPartitionOffer) []yacymodel.Hash {
	peers := make([]yacymodel.Hash, 0, len(offer.Recipients))
	for _, recipient := range offer.Recipients {
		peers = append(peers, recipient.Peer.Hash)
	}

	return peers
}

func postingsSentTo(
	t *testing.T,
	offer wordpartitionoffer.WordPartitionOffer,
	peer yacymodel.Seed,
) []yacymodel.RWIPosting {
	t.Helper()

	for _, recipient := range offer.Recipients {
		if recipient.Peer.Hash == peer.Hash {
			return recipient.Postings
		}
	}
	t.Fatalf("recipients = %v, want %v among them", recipientPeersOf(offer), peer.Hash)

	return nil
}

func urlsSentTo(
	t *testing.T,
	offer wordpartitionoffer.WordPartitionOffer,
	peer yacymodel.Seed,
) []yacymodel.URLHash {
	t.Helper()

	postings := postingsSentTo(t, offer, peer)
	urls := make([]yacymodel.URLHash, 0, len(postings))
	for _, posting := range postings {
		urls = append(urls, posting.URLHash)
	}

	return sortedURLs(urls)
}

func undeliveredURLsOf(offer wordpartitionoffer.WordPartitionOffer) []yacymodel.URLHash {
	urls := make([]yacymodel.URLHash, 0, len(offer.UndeliveredPostings))
	for _, posting := range offer.UndeliveredPostings {
		urls = append(urls, posting.URL)
	}

	return sortedURLs(urls)
}

func sortedURLs(urls []yacymodel.URLHash) []yacymodel.URLHash {
	return slices.SortedFunc(slices.Values(urls), func(url, other yacymodel.URLHash) int {
		return cmp.Compare(url.String(), other.String())
	})
}

type fakeReachability struct {
	reachable         []yacymodel.Seed
	recentlyReachable map[yacymodel.Hash]bool
}

func (f *fakeReachability) IsReachable(_ context.Context, peer yacymodel.Hash) bool {
	return slices.ContainsFunc(f.reachable, func(seed yacymodel.Seed) bool {
		return seed.Hash == peer
	})
}

func (f *fakeReachability) IsRecentlyReachable(_ context.Context, peer yacymodel.Hash) bool {
	return f.recentlyReachable[peer]
}

type fakeEligibility struct {
	cooldowns     map[yacymodel.Hash]time.Duration
	excludedPeers map[yacymodel.Hash]bool
}

func (f fakeEligibility) CooldownLeftOf(peer yacymodel.Hash) time.Duration {
	return f.cooldowns[peer]
}

func (f fakeEligibility) IsExcluded(peer yacymodel.Hash) bool {
	return f.excludedPeers[peer]
}

type fakeObserver struct {
	peersPerSector []int
}

func (f *fakeObserver) ObservePeersAcceptingRemoteIndexPerDHTRingSector(peersPerSector []int) {
	f.peersPerSector = peersPerSector
}

func TestAHolderGetsOnlyTheUndeliveredPostingsAndAMissingHolderTheWholeWordPartition(
	t *testing.T,
) {
	h := openHarness(t, options{redundancy: 2})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	urls := urlsOfPartition(3)
	holder, missingHolder := peerAfter(wordPartition, 0), peerAfter(wordPartition, 1)
	h.admit(t, wordPartition, urls[:2]...)
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, holder.Hash)
	h.admit(t, wordPartition, urls[2])

	offers, _ := h.dueNow(t, wordpartitionoffer.Cursor{}, holder, missingHolder)

	offer := onlyOffer(t, offers)
	if got, want := urlsSentTo(t, offer, holder), urls[2:]; !slices.Equal(got, want) {
		t.Fatalf("holder gets %v, want only the undelivered %v", got, want)
	}
	if got, want := urlsSentTo(
		t,
		offer,
		missingHolder,
	), sortedURLs(
		urls,
	); !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("missing holder gets %v, want the whole word partition %v", got, want)
	}
	if got, want := undeliveredURLsOf(offer), urls[2:]; !slices.Equal(got, want) {
		t.Fatalf("UndeliveredPostings = %v, want %v", got, want)
	}
}

func TestAWordPartitionWithARecipientInCooldownWaitsAndTheWalkMovesPastIt(t *testing.T) {
	h := openHarness(t, options{postingsPerBatch: 1})
	wordPartitions := wordPartitionsInRingOrder(t, 2)
	waiting, free := wordPartitions[0], wordPartitions[1]
	peerInCooldown, peerOutOfCooldown := peerAfter(waiting, 0), peerAfter(free, 0)
	h.admit(t, waiting, urlsOfPartition(1)...)
	h.admit(t, free, urlsOfPartition(1)...)
	h.eligibility.cooldowns[peerInCooldown.Hash] = time.Minute

	offers, next := h.dueNow(t, wordpartitionoffer.Cursor{}, peerInCooldown, peerOutOfCooldown)
	if got := wordPartitionsOf(offers); !slices.Equal(got, []yacymodel.WordPartition{free}) {
		t.Fatalf("offers = %v, want only %v while %v waits", got, free, waiting)
	}

	delete(h.eligibility.cooldowns, peerInCooldown.Hash)
	offers, next = h.dueNow(t, next, peerInCooldown, peerOutOfCooldown)
	if len(offers) != 0 || next != (wordpartitionoffer.Cursor{}) {
		t.Fatalf("offers after the cursor = %v, want none and the sector end", offers)
	}
}

func TestEachCallContinuesFromTheCursorAndWrapsAtTheSectorEnd(t *testing.T) {
	h := openHarness(t, options{postingsPerBatch: 1})
	wordPartitions := wordPartitionsInRingOrder(t, 3)
	peer := peerAfter(wordPartitions[2], 1)
	for _, wordPartition := range wordPartitions {
		h.admit(t, wordPartition, urlsOfPartition(1)...)
	}

	var cursor wordpartitionoffer.Cursor
	for _, wordPartition := range wordPartitions {
		var offers []wordpartitionoffer.WordPartitionOffer
		offers, cursor = h.dueNow(t, cursor, peer)
		if got := onlyOffer(t, offers).WordPartition; got != wordPartition {
			t.Fatalf("offer = %v, want %v", got, wordPartition)
		}
	}
	offers, cursor := h.dueNow(t, cursor, peer)
	if len(offers) != 0 || cursor != (wordpartitionoffer.Cursor{}) {
		t.Fatalf("offers at the sector end = %v, want none and the sector start", offers)
	}
	offers, _ = h.dueNow(t, cursor, peer)
	if got := onlyOffer(t, offers).WordPartition; got != wordPartitions[0] {
		t.Fatalf("offer after the wrap = %v, want %v", got, wordPartitions[0])
	}
}

func TestTheReadBudgetEndsABatchWhereEveryWordPartitionWaits(t *testing.T) {
	const postingsPerBatch = 1
	h := openHarness(t, options{postingsPerBatch: postingsPerBatch})
	wordPartitions := wordPartitionsInRingOrder(t, 12)
	peer := peerAfter(wordPartitions[len(wordPartitions)-1], 1)
	for _, wordPartition := range wordPartitions {
		h.admit(t, wordPartition, urlsOfPartition(1)...)
	}
	h.eligibility.cooldowns[peer.Hash] = time.Minute

	offers, next := h.dueNow(t, wordpartitionoffer.Cursor{}, peer)
	if len(offers) != 0 || next == (wordpartitionoffer.Cursor{}) {
		t.Fatalf("offers = %v, want none and a batch that ends before the sector end", offers)
	}
	offers, next = h.dueNow(t, next, peer)
	if len(offers) != 0 || next != (wordpartitionoffer.Cursor{}) {
		t.Fatalf("offers = %v, want none and the sector end", offers)
	}
}

func TestAWordPartitionIsNeverSplitAcrossBatches(t *testing.T) {
	h := openHarness(t, options{postingsPerBatch: 2})
	wordPartitions := wordPartitionsInRingOrder(t, 2)
	peer := peerAfter(wordPartitions[1], 1)
	urls := urlsOfPartition(3)
	h.admit(t, wordPartitions[0], urls...)
	h.admit(t, wordPartitions[1], urls[0])

	offers, next := h.dueNow(t, wordpartitionoffer.Cursor{}, peer)
	offer := onlyOffer(t, offers)
	if offer.WordPartition != wordPartitions[0] ||
		!slices.Equal(undeliveredURLsOf(offer), sortedURLs(urls)) {
		t.Fatalf("offer = %+v, want all of %v", offer, wordPartitions[0])
	}
	offers, _ = h.dueNow(t, next, peer)
	if got := onlyOffer(t, offers).WordPartition; got != wordPartitions[1] {
		t.Fatalf("next offer = %v, want %v", got, wordPartitions[1])
	}
}

func TestAMissingHolderGetsTheWholeOfABigWordPartitionInOneOffer(t *testing.T) {
	h := openHarness(t, options{postingsPerBatch: 2})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	missingHolder := peerAfter(wordPartition, 0)
	urls := urlsOfPartition(5)
	h.admit(t, wordPartition, urls...)

	offer := onlyOffer(t, withoutCursor(h.dueNow(t, wordpartitionoffer.Cursor{}, missingHolder)))

	if got := urlsSentTo(t, offer, missingHolder); !slices.Equal(got, sortedURLs(urls)) {
		t.Fatalf("missing holder gets %v, want the whole word partition %v", got, urls)
	}
}

func TestDueRefreshesSendTheWholeWordPartitionAfterTheUndeliveredOnes(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	wordPartitions := wordPartitionsInRingOrder(t, 2)
	refreshed, undelivered := wordPartitions[0], wordPartitions[1]
	holder, missingHolder := peerAfter(refreshed, 0), peerAfter(refreshed, 1)
	urls := urlsOfPartition(2)
	h.admit(t, refreshed, urls...)
	h.deliverEverything(t)
	h.recordHolders(t, refreshed, holder.Hash)
	h.recordHolders(t, undelivered, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)
	h.admit(t, undelivered, urls[0])

	offers, _ := h.dueNow(t, wordpartitionoffer.Cursor{}, holder, missingHolder)

	if got := wordPartitionsOf(offers); !slices.Equal(
		got,
		[]yacymodel.WordPartition{undelivered, refreshed},
	) {
		t.Fatalf("offers = %v, want %v and then the refresh of %v", got, undelivered, refreshed)
	}
	refresh := offers[1]
	if !refresh.IsRefresh || offers[0].IsRefresh {
		t.Fatalf("refreshes = %v, %v, want only the second", offers[0].IsRefresh, refresh.IsRefresh)
	}
	for _, peer := range []yacymodel.Seed{holder, missingHolder} {
		if got := urlsSentTo(t, refresh, peer); !slices.Equal(got, sortedURLs(urls)) {
			t.Fatalf("%v gets %v, want the whole word partition %v", peer.Hash, got, urls)
		}
	}
}

func TestADueRefreshCarriesTheUndeliveredPostingsOfItsWordPartition(t *testing.T) {
	h := openHarness(t, options{postingsPerBatch: 1})
	wordPartitions := wordPartitionsInRingOrder(t, 2)
	refreshed, other := wordPartitions[0], wordPartitions[1]
	holder, otherPeer := peerAfter(refreshed, 0), peerAfter(other, 0)
	urls := urlsOfPartition(3)
	h.admit(t, refreshed, urls[:2]...)
	h.deliverEverything(t)
	h.recordHolders(t, refreshed, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)
	h.admit(t, refreshed, urls[2])
	h.admit(t, other, urls[0])
	h.eligibility.cooldowns[holder.Hash] = time.Minute
	_, next := h.dueNow(t, wordpartitionoffer.Cursor{}, holder, otherPeer)
	delete(h.eligibility.cooldowns, holder.Hash)

	offers, _ := h.dueNow(t, next, holder, otherPeer)

	refresh := onlyOffer(t, offers)
	if !refresh.IsRefresh || refresh.WordPartition != refreshed {
		t.Fatalf("offer = %+v, want the refresh of %v", refresh, refreshed)
	}
	if got, want := undeliveredURLsOf(refresh), urls[2:]; !slices.Equal(got, want) {
		t.Fatalf("UndeliveredPostings = %v, want %v", got, want)
	}
	if got := urlsSentTo(t, refresh, holder); !slices.Equal(got, sortedURLs(urls)) {
		t.Fatalf("holder gets %v, want the whole word partition %v", got, urls)
	}
}

func TestADueRefreshWithoutHoldersSendsTheWholeWordPartitionToEveryResponsiblePeer(
	t *testing.T,
) {
	h := openHarness(t, options{redundancy: 2})
	refreshed := wordPartitionsInRingOrder(t, 1)[0]
	responsiblePeers := []yacymodel.Seed{peerAfter(refreshed, 0), peerAfter(refreshed, 1)}
	urls := urlsOfPartition(2)
	h.admit(t, refreshed, urls...)
	h.deliverEverything(t)
	h.recordNoHolders(t, refreshed)
	h.clock = h.clock.Add(retryDelay)

	refresh := onlyOffer(
		t,
		withoutCursor(h.dueNow(t, wordpartitionoffer.Cursor{}, responsiblePeers...)),
	)

	if !refresh.IsRefresh || len(refresh.Holders) != 0 {
		t.Fatalf("offer = %+v, want a refresh without holders", refresh)
	}
	for _, peer := range responsiblePeers {
		if got := urlsSentTo(t, refresh, peer); !slices.Equal(got, sortedURLs(urls)) {
			t.Fatalf("%v gets %v, want the whole word partition %v", peer.Hash, got, urls)
		}
	}
}

func TestDueRefreshesStopOnceTheBatchIsFull(t *testing.T) {
	h := openHarness(t, options{postingsPerBatch: 2})
	wordPartitions := wordPartitionsInRingOrder(t, 2)
	holder := peerAfter(wordPartitions[1], 1)
	for _, wordPartition := range wordPartitions {
		h.admit(t, wordPartition, urlsOfPartition(2)...)
		h.recordHolders(t, wordPartition, holder.Hash)
	}
	h.deliverEverything(t)
	h.clock = h.clock.Add(refreshInterval)

	offers, _ := h.dueNow(t, wordpartitionoffer.Cursor{}, holder)

	if !onlyOffer(t, offers).IsRefresh {
		t.Fatalf("offer = %+v, want a refresh", offers[0])
	}
}

func TestThisNodeAmongTheResponsiblePeersIsNeverARecipient(t *testing.T) {
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	h := openHarness(t, options{
		self:       yacymodel.HashFromDHTRingPosition(positionOf(wordPartition)),
		redundancy: 2,
	})
	closest, farther := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
	h.admit(t, wordPartition, urlsOfPartition(1)...)

	offer := onlyOffer(t, withoutCursor(h.dueNow(t, wordpartitionoffer.Cursor{}, closest, farther)))

	want := []yacymodel.Hash{closest.Hash}
	if !slices.Equal(offer.ResponsiblePeers, want) || !slices.Equal(recipientPeersOf(offer), want) {
		t.Fatalf(
			"responsible peers %v and recipients %v, want only %v: this node holds the other copy",
			offer.ResponsiblePeers, recipientPeersOf(offer), want,
		)
	}
}

func TestAWordPartitionWithoutRecipientsWaits(t *testing.T) {
	h := openHarness(t, options{postingsPerBatch: 1})
	wordPartitions := wordPartitionsInRingOrder(t, 2)
	waiting, free := wordPartitions[0], wordPartitions[1]
	responsiblePeer, otherPeer := peerAfter(waiting, 0), peerAfter(free, 0)
	absentHolder := peerAfter(waiting, 1<<40)
	h.admit(t, waiting, urlsOfPartition(1)...)
	h.admit(t, free, urlsOfPartition(1)...)
	h.recordHolders(t, waiting, absentHolder.Hash)
	h.reachability.recentlyReachable[absentHolder.Hash] = true

	offers, _ := h.dueNow(t, wordpartitionoffer.Cursor{}, responsiblePeer, otherPeer)

	if got := wordPartitionsOf(offers); !slices.Equal(got, []yacymodel.WordPartition{free}) {
		t.Fatalf(
			"offers = %v, want only %v: the recently reachable holder of %v is not replaced",
			got, free, waiting,
		)
	}
}

func TestAHolderOutsideTheResponsiblePeersIsStaleAndReplaced(t *testing.T) {
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	responsiblePeer := peerAfter(wordPartition, 0)
	holder := peerAfter(wordPartition, 1<<40)
	for name, holderReachable := range map[string]bool{
		"expired holder":                      false,
		"holder beyond the responsible peers": true,
	} {
		t.Run(name, func(t *testing.T) {
			h := openHarness(t, options{})
			h.admit(t, wordPartition, urlsOfPartition(1)...)
			h.recordHolders(t, wordPartition, holder.Hash)
			reachablePeers := []yacymodel.Seed{responsiblePeer}
			if holderReachable {
				reachablePeers = append(reachablePeers, holder)
			}

			offer := onlyOffer(
				t,
				withoutCursor(h.dueNow(t, wordpartitionoffer.Cursor{}, reachablePeers...)),
			)

			want := []yacymodel.Hash{responsiblePeer.Hash}
			if got := recipientPeersOf(offer); !slices.Equal(got, want) {
				t.Fatalf("recipients = %v, want %v", got, want)
			}
			if want := []yacymodel.Hash{holder.Hash}; !slices.Equal(offer.StaleHolders, want) {
				t.Fatalf("StaleHolders = %v, want %v", offer.StaleHolders, want)
			}
		})
	}
}

func withoutCursor(
	offers []wordpartitionoffer.WordPartitionOffer,
	_ wordpartitionoffer.Cursor,
) []wordpartitionoffer.WordPartitionOffer {
	return offers
}

func TestReportCountsOnlyThePeersAcceptingRemoteIndexPerSector(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	accepting, refusing := peerAfter(wordPartition, 0), peerAfter(wordPartition, 1)
	refusing.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{})

	h.offers.ReportPeersAcceptingRemoteIndex([]yacymodel.Seed{accepting, refusing})

	sector := yacymodel.DHTRingSectorOf(yacymodel.DHTRingPositionOf(accepting.Hash))
	sumOfPeers := 0
	for _, peers := range h.observer.peersPerSector {
		sumOfPeers += peers
	}
	if sumOfPeers != 1 || h.observer.peersPerSector[sector] != 1 {
		t.Fatalf("peers per sector = %v, want only %v in sector %d",
			h.observer.peersPerSector, accepting.Hash, sector)
	}
}

func TestAWordPartitionInRetryGoesOutOnlyAsADueRefresh(t *testing.T) {
	h := openHarness(t, options{})
	inRetry := wordPartitionsInRingOrder(t, 1)[0]
	peer := peerAfter(inRetry, 0)
	urls := urlsOfPartition(1)
	h.admit(t, inRetry, urls...)
	h.recordNoHolders(t, inRetry)

	offers, next := h.dueNow(t, wordpartitionoffer.Cursor{}, peer)
	if len(offers) != 0 || next != (wordpartitionoffer.Cursor{}) {
		t.Fatalf("offers before the retry = %v, want none and the sector end", offers)
	}

	h.clock = h.clock.Add(retryDelay)
	refresh := onlyOffer(t, withoutCursor(h.dueNow(t, wordpartitionoffer.Cursor{}, peer)))

	if !refresh.IsRefresh {
		t.Fatalf("offer = %+v, want the retry as a refresh", refresh)
	}
	if got := undeliveredURLsOf(refresh); !slices.Equal(got, urls) {
		t.Fatalf("UndeliveredPostings = %v, want %v", got, urls)
	}
}

func TestAnExcludedHolderIsStaleAndTheNextPeerGetsTheWholeWordPartition(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	urls := urlsOfPartition(2)
	excludedHolder := peerAfter(wordPartition, 0)
	holder, nextPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
	h.admit(t, wordPartition, urls[0])
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, excludedHolder.Hash, holder.Hash)
	h.admit(t, wordPartition, urls[1])
	h.eligibility.excludedPeers[excludedHolder.Hash] = true

	offer := onlyOffer(
		t,
		withoutCursor(h.dueNow(t, wordpartitionoffer.Cursor{}, excludedHolder, holder, nextPeer)),
	)

	if got, want := recipientPeersOf(
		offer,
	), []yacymodel.Hash{
		holder.Hash,
		nextPeer.Hash,
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("recipients = %v, want %v", got, want)
	}
	if got, want := urlsSentTo(t, offer, nextPeer), sortedURLs(urls); !slices.Equal(got, want) {
		t.Fatalf("next peer gets %v, want the whole word partition %v", got, want)
	}
	if want := []yacymodel.Hash{excludedHolder.Hash}; !slices.Equal(offer.StaleHolders, want) {
		t.Fatalf("StaleHolders = %v, want %v", offer.StaleHolders, want)
	}
	if !slices.Equal(offer.ResponsiblePeers, []yacymodel.Hash{holder.Hash, nextPeer.Hash}) {
		t.Fatalf("ResponsiblePeers = %v, want the holder and the next peer", offer.ResponsiblePeers)
	}
}

func TestAnUndeliveredWordPartitionWithARecipientInCooldownIsSkipped(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	peer := peerAfter(wordPartition, 0)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.eligibility.cooldowns[peer.Hash] = time.Minute

	batch := h.dueBatch(t, wordpartitionoffer.Cursor{}, peer)

	assertSkips(t, batch.Skips,
		wordpartitionoffer.UndeliveredOffer,
		wordpartitionoffer.SkipRecipientInCooldown,
	)
}

func TestAnUndeliveredWordPartitionInRetryIsSkipped(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.recordNoHolders(t, wordPartition)

	batch := h.dueBatch(t, wordpartitionoffer.Cursor{}, peerAfter(wordPartition, 0))

	assertSkips(t, batch.Skips,
		wordpartitionoffer.UndeliveredOffer,
		wordpartitionoffer.SkipWordPartitionInRetry,
	)
}

func TestAnUndeliveredWordPartitionWithoutRecipientsIsSkipped(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	absentHolder := peerAfter(wordPartition, 1<<40)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.recordHolders(t, wordPartition, absentHolder.Hash)
	h.reachability.recentlyReachable[absentHolder.Hash] = true

	batch := h.dueBatch(t, wordpartitionoffer.Cursor{}, peerAfter(wordPartition, 0))

	assertSkips(
		t,
		batch.Skips,
		wordpartitionoffer.UndeliveredOffer,
		wordpartitionoffer.SkipNoRecipient,
	)
}

func TestADueRefreshWithARecipientInCooldownIsSkipped(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	holder := peerAfter(wordPartition, 0)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)
	h.eligibility.cooldowns[holder.Hash] = time.Minute

	batch := h.dueBatch(t, wordpartitionoffer.Cursor{}, holder)

	assertSkips(
		t,
		batch.Skips,
		wordpartitionoffer.RefreshOffer,
		wordpartitionoffer.SkipRecipientInCooldown,
	)
}

func TestADueRefreshWithoutRecipientsIsSkipped(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	absentHolder := peerAfter(wordPartition, 1<<40)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, absentHolder.Hash)
	h.reachability.recentlyReachable[absentHolder.Hash] = true
	h.clock = h.clock.Add(refreshInterval)

	batch := h.dueBatch(t, wordpartitionoffer.Cursor{}, peerAfter(wordPartition, 0))

	assertSkips(
		t,
		batch.Skips,
		wordpartitionoffer.RefreshOffer,
		wordpartitionoffer.SkipNoRecipient,
	)
}

func TestAnOfferedWordPartitionIsNotSkipped(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := wordPartitionsInRingOrder(t, 1)[0]
	holder := peerAfter(wordPartition, 0)
	h.admit(t, wordPartition, urlsOfPartition(2)...)
	h.recordHolders(t, wordPartition, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)

	batch := h.dueBatch(t, wordpartitionoffer.Cursor{}, holder)

	if len(batch.Offers) == 0 || len(batch.Skips) != 0 {
		t.Fatalf("offers = %v and skips = %v, want an offer and no skip", batch.Offers, batch.Skips)
	}
}

func assertSkips(
	t *testing.T,
	skips []wordpartitionoffer.Skip,
	offerKind wordpartitionoffer.OfferKind,
	skipReason wordpartitionoffer.SkipReason,
) {
	t.Helper()

	want := []wordpartitionoffer.Skip{
		{Offer: offerKind, Reason: skipReason},
	}
	if !slices.Equal(skips, want) {
		t.Fatalf("skips = %v, want %v", skips, want)
	}
}
