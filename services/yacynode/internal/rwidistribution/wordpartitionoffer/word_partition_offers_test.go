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
	partitionExponent         = 2
	refreshInterval           = time.Hour
	retryDelay                = time.Minute
	positionStep              = 8
	everyEntryBudget          = 1 << 20
	mostWordPartitionsInATest = 2
)

type options struct {
	self       yacymodel.Hash
	redundancy int
}

type harness struct {
	vault        *vault.Vault
	admitter     rwipostings.PostingAdmitter
	undelivered  *undeliveredpostings.UndeliveredPostings
	holders      *wordpartitionholders.Holders
	offers       *wordpartitionoffer.WordPartitionOffers
	reachability *fakeReachability
	eligibility  fakeEligibility
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
			peersInCooldown: map[yacymodel.Hash]bool{},
			excludedPeers:   map[yacymodel.Hash]bool{},
		},
		clock: time.Unix(1_700_000_000, 0).UTC(),
	}
	h.holders, err = wordpartitionholders.Open(v, partitions(), refreshInterval, retryDelay, h.now)
	if err != nil {
		t.Fatalf("wordpartitionholders.Open: %v", err)
	}
	h.offers = wordpartitionoffer.New(
		v, undelivered, h.holders, index, h.reachability, h.eligibility, configFrom(opts),
	)

	return h
}

func configFrom(opts options) wordpartitionoffer.Config {
	config := wordpartitionoffer.Config{
		Partitions: partitions(),
		Self:       opts.self,
		Redundancy: opts.redundancy,
	}
	if config.Self.IsZero() {
		config.Self = selfFartherThanThePeersOfEveryWordPartition()
	}
	if config.Redundancy == 0 {
		config.Redundancy = 1
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

	undelivered := h.undeliveredWordPartitions(t)
	h.write(t, func(tx *vault.Txn) error {
		for _, wordPartition := range undelivered {
			for _, posting := range wordPartition.Postings {
				if _, err := h.undelivered.MarkDelivered(tx, posting); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (h *harness) undeliveredWordPartitions(
	t *testing.T,
) []undeliveredpostings.UndeliveredWordPartition {
	t.Helper()

	var wordPartitions []undeliveredpostings.UndeliveredWordPartition
	h.read(t, func(tx *vault.Txn) error {
		var err error
		wordPartitions, _, err = h.undelivered.WordPartitionsAfter(
			tx, undeliveredpostings.UndeliveredPosting{}, everyEntryBudget)

		return err
	})

	return wordPartitions
}

func (h *harness) dueWordPartitions(t *testing.T) []yacymodel.WordPartition {
	t.Helper()

	var dueWordPartitions []yacymodel.WordPartition
	h.read(t, func(tx *vault.Txn) error {
		dueRefreshes, _, err := h.holders.DueRefreshesAfter(
			tx, wordpartitionholders.ScheduledRefresh{}, everyEntryBudget)
		for _, dueRefresh := range dueRefreshes {
			dueWordPartitions = append(dueWordPartitions, dueRefresh.WordPartition)
		}

		return err
	})

	return dueWordPartitions
}

func (h *harness) recordHolders(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
	recordedHolders ...yacymodel.Hash,
) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.RecordAtRedundancy(tx, wordPartition, recordedHolders)
	})
}

func (h *harness) recordNoHolders(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.holders.RecordShortOfRedundancy(tx, wordPartition, nil)
	})
}

func (h *harness) undeliveredBatch(
	t *testing.T,
	reachablePeers ...yacymodel.Seed,
) wordpartitionoffer.Batch {
	t.Helper()

	undelivered := h.undeliveredWordPartitions(t)
	h.reachability.reachable = reachablePeers
	batch, err := h.offers.UndeliveredOffersFor(context.Background(), undelivered, reachablePeers)
	if err != nil {
		t.Fatalf("UndeliveredOffersFor: %v", err)
	}

	return batch
}

func (h *harness) refreshBatch(
	t *testing.T,
	reachablePeers ...yacymodel.Seed,
) wordpartitionoffer.Batch {
	t.Helper()

	due := h.dueWordPartitions(t)
	h.reachability.reachable = reachablePeers
	batch, err := h.offers.RefreshOffersFor(context.Background(), due, reachablePeers)
	if err != nil {
		t.Fatalf("RefreshOffersFor: %v", err)
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

func positionOf(wordPartition yacymodel.WordPartition) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOfWordPartition(wordPartition, partitions())
}

func fixtureWordPartitions(amount int) []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, amount)
	for word := range amount {
		wordPartitions = append(
			wordPartitions,
			yacymodel.WordPartition{Word: wordOf(word), Partition: fixturePartition()},
		)
	}

	return wordPartitions
}

func selfFartherThanThePeersOfEveryWordPartition() yacymodel.Hash {
	positions := make([]yacymodel.DHTRingPosition, 0, mostWordPartitionsInATest)
	for _, wordPartition := range fixtureWordPartitions(mostWordPartitionsInATest) {
		positions = append(positions, positionOf(wordPartition))
	}

	return yacymodel.HashFromDHTRingPosition(slices.Min(positions) - 1<<50)
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
		wordPartitions = append(wordPartitions, offer.OutcomeFrom(nil).WordPartition)
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
	return urlsOfUndelivered(offer.OutcomeFrom(everythingAcceptedIn(offer)).DeliveredPostings)
}

func everythingAcceptedIn(
	offer wordpartitionoffer.WordPartitionOffer,
) map[yacymodel.Hash]wordpartitionoffer.PeerAnswer {
	answers := make(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer, len(offer.Recipients))
	for _, recipient := range offer.Recipients {
		answers[recipient.Peer.Hash] = accepted(recipient.Postings)
	}

	return answers
}

func urlsOfUndelivered(postings []undeliveredpostings.UndeliveredPosting) []yacymodel.URLHash {
	urls := make([]yacymodel.URLHash, 0, len(postings))
	for _, posting := range postings {
		urls = append(urls, posting.URL)
	}

	return sortedURLs(urls)
}

func sortedURLs(urls []yacymodel.URLHash) []yacymodel.URLHash {
	return slices.SortedFunc(slices.Values(urls), func(url, other yacymodel.URLHash) int {
		return cmp.Compare(url.String(), other.String())
	})
}

func assertOnlyStaleHolder(
	t *testing.T,
	offer wordpartitionoffer.WordPartitionOffer,
	staleHolder yacymodel.Seed,
) {
	t.Helper()

	outcome := offer.OutcomeFrom(nil)
	if outcome.AmountOfStaleHolders != 1 || slices.Contains(outcome.Holders, staleHolder.Hash) {
		t.Fatalf("AmountOfStaleHolders = %d with holders %v, want only %v stale",
			outcome.AmountOfStaleHolders, outcome.Holders, staleHolder.Hash)
	}
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
	peersInCooldown map[yacymodel.Hash]bool
	excludedPeers   map[yacymodel.Hash]bool
}

func (f fakeEligibility) IsInCooldown(peer yacymodel.Hash) bool {
	return f.peersInCooldown[peer]
}

func (f fakeEligibility) IsExcluded(peer yacymodel.Hash) bool {
	return f.excludedPeers[peer]
}

func TestAHolderGetsOnlyTheUndeliveredPostingsAndAMissingHolderTheWholeWordPartition(
	t *testing.T,
) {
	h := openHarness(t, options{redundancy: 2})
	wordPartition := fixtureWordPartitions(1)[0]
	urls := urlsOfPartition(3)
	holder, missingHolder := peerAfter(wordPartition, 0), peerAfter(wordPartition, 1)
	h.admit(t, wordPartition, urls[:2]...)
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, holder.Hash)
	h.admit(t, wordPartition, urls[2])

	offers := h.undeliveredBatch(t, holder, missingHolder).Offers

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

func TestAWordPartitionWithARecipientInCooldownWaitsWhileTheOthersGoOut(t *testing.T) {
	h := openHarness(t, options{})
	wordPartitions := fixtureWordPartitions(2)
	waiting, free := wordPartitions[0], wordPartitions[1]
	peerInCooldown, peerOutOfCooldown := peerAfter(waiting, 0), peerAfter(free, 0)
	h.admit(t, waiting, urlsOfPartition(1)...)
	h.admit(t, free, urlsOfPartition(1)...)
	h.eligibility.peersInCooldown[peerInCooldown.Hash] = true

	offers := h.undeliveredBatch(t, peerInCooldown, peerOutOfCooldown).Offers

	if got := wordPartitionsOf(offers); !slices.Equal(got, []yacymodel.WordPartition{free}) {
		t.Fatalf("offers = %v, want only %v while %v waits", got, free, waiting)
	}
}

func TestAMissingHolderGetsTheWholeOfABigWordPartitionInOneOffer(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	missingHolder := peerAfter(wordPartition, 0)
	urls := urlsOfPartition(5)
	h.admit(t, wordPartition, urls...)

	offer := onlyOffer(t, h.undeliveredBatch(t, missingHolder).Offers)

	if got := urlsSentTo(t, offer, missingHolder); !slices.Equal(got, sortedURLs(urls)) {
		t.Fatalf("missing holder gets %v, want the whole word partition %v", got, urls)
	}
}

func TestADueRefreshSendsTheWholeWordPartitionToEveryRecipient(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	wordPartitions := fixtureWordPartitions(2)
	refreshed, notDue := wordPartitions[0], wordPartitions[1]
	holder, missingHolder := peerAfter(refreshed, 0), peerAfter(refreshed, 1)
	urls := urlsOfPartition(2)
	h.admit(t, refreshed, urls...)
	h.deliverEverything(t)
	h.recordHolders(t, refreshed, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)
	h.recordHolders(t, notDue, holder.Hash)

	refresh := onlyOffer(t, h.refreshBatch(t, holder, missingHolder).Offers)

	if refresh.OutcomeFrom(nil).WordPartition != refreshed {
		t.Fatalf("offer = %+v, want the refresh of %v", refresh, refreshed)
	}
	for _, peer := range []yacymodel.Seed{holder, missingHolder} {
		if got := urlsSentTo(t, refresh, peer); !slices.Equal(got, sortedURLs(urls)) {
			t.Fatalf("%v gets %v, want the whole word partition %v", peer.Hash, got, urls)
		}
	}
}

func TestADueRefreshCarriesTheUndeliveredPostingsOfItsWordPartition(t *testing.T) {
	h := openHarness(t, options{})
	refreshed := fixtureWordPartitions(1)[0]
	holder := peerAfter(refreshed, 0)
	urls := urlsOfPartition(3)
	h.admit(t, refreshed, urls[:2]...)
	h.deliverEverything(t)
	h.recordHolders(t, refreshed, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)
	h.admit(t, refreshed, urls[2])

	refresh := onlyOffer(t, h.refreshBatch(t, holder).Offers)

	if refresh.OutcomeFrom(nil).WordPartition != refreshed {
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
	refreshed := fixtureWordPartitions(1)[0]
	responsiblePeers := []yacymodel.Seed{peerAfter(refreshed, 0), peerAfter(refreshed, 1)}
	urls := urlsOfPartition(2)
	h.admit(t, refreshed, urls...)
	h.deliverEverything(t)
	h.recordNoHolders(t, refreshed)
	h.clock = h.clock.Add(retryDelay)

	refresh := onlyOffer(t, h.refreshBatch(t, responsiblePeers...).Offers)

	if len(refresh.OutcomeFrom(nil).Holders) != 0 {
		t.Fatalf("offer = %+v, want a refresh without holders", refresh)
	}
	for _, peer := range responsiblePeers {
		if got := urlsSentTo(t, refresh, peer); !slices.Equal(got, sortedURLs(urls)) {
			t.Fatalf("%v gets %v, want the whole word partition %v", peer.Hash, got, urls)
		}
	}
}

func TestThisNodeAmongTheResponsiblePeersIsNeverARecipient(t *testing.T) {
	wordPartition := fixtureWordPartitions(1)[0]
	h := openHarness(t, options{
		self:       yacymodel.HashFromDHTRingPosition(positionOf(wordPartition)),
		redundancy: 2,
	})
	closest, farther := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
	h.admit(t, wordPartition, urlsOfPartition(1)...)

	offer := onlyOffer(t, h.undeliveredBatch(t, closest, farther).Offers)

	want := []yacymodel.Hash{closest.Hash}
	responsiblePeers := offer.OutcomeFrom(nil).ResponsiblePeers
	if !slices.Equal(responsiblePeers, want) || !slices.Equal(recipientPeersOf(offer), want) {
		t.Fatalf(
			"responsible peers %v and recipients %v, want only %v: this node holds the other copy",
			responsiblePeers, recipientPeersOf(offer), want,
		)
	}
}

func TestAWordPartitionWithoutRecipientsWaits(t *testing.T) {
	h := openHarness(t, options{})
	wordPartitions := fixtureWordPartitions(2)
	waiting, free := wordPartitions[0], wordPartitions[1]
	responsiblePeer, otherPeer := peerAfter(waiting, 0), peerAfter(free, 0)
	absentHolder := peerAfter(waiting, 1<<40)
	h.admit(t, waiting, urlsOfPartition(1)...)
	h.admit(t, free, urlsOfPartition(1)...)
	h.recordHolders(t, waiting, absentHolder.Hash)
	h.reachability.recentlyReachable[absentHolder.Hash] = true

	offers := h.undeliveredBatch(t, responsiblePeer, otherPeer).Offers

	if got := wordPartitionsOf(offers); !slices.Equal(got, []yacymodel.WordPartition{free}) {
		t.Fatalf(
			"offers = %v, want only %v: the recently reachable holder of %v is not replaced",
			got, free, waiting,
		)
	}
}

func TestAHolderOutsideTheResponsiblePeersIsStaleAndReplaced(t *testing.T) {
	wordPartition := fixtureWordPartitions(1)[0]
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

			offer := onlyOffer(t, h.undeliveredBatch(t, reachablePeers...).Offers)

			want := []yacymodel.Hash{responsiblePeer.Hash}
			if got := recipientPeersOf(offer); !slices.Equal(got, want) {
				t.Fatalf("recipients = %v, want %v", got, want)
			}
			assertOnlyStaleHolder(t, offer, holder)
		})
	}
}

func TestAPeerRefusingRemoteIndexIsNeverARecipient(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	refusingPeer, acceptingPeer := peerAfter(wordPartition, 0), peerAfter(wordPartition, 1)
	refusingPeer.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{})
	h.admit(t, wordPartition, urlsOfPartition(1)...)

	offer := onlyOffer(t, h.undeliveredBatch(t, refusingPeer, acceptingPeer).Offers)

	if got, want := recipientPeersOf(
		offer,
	), []yacymodel.Hash{
		acceptingPeer.Hash,
	}; !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("recipients = %v, want only %v", got, want)
	}
}

func TestAWordPartitionInRetryGoesOutOnlyAsADueRefresh(t *testing.T) {
	h := openHarness(t, options{})
	inRetry := fixtureWordPartitions(1)[0]
	peer := peerAfter(inRetry, 0)
	urls := urlsOfPartition(1)
	h.admit(t, inRetry, urls...)
	h.recordNoHolders(t, inRetry)

	if offers := h.undeliveredBatch(t, peer).Offers; len(offers) != 0 {
		t.Fatalf("offers before the retry = %v, want none", offers)
	}

	h.clock = h.clock.Add(retryDelay)
	if offers := h.undeliveredBatch(t, peer).Offers; len(offers) != 0 {
		t.Fatalf("undelivered offers at the retry = %v, want none", offers)
	}
	refresh := onlyOffer(t, h.refreshBatch(t, peer).Offers)

	if got := undeliveredURLsOf(refresh); !slices.Equal(got, urls) {
		t.Fatalf("UndeliveredPostings = %v, want %v", got, urls)
	}
}

func TestAnExcludedHolderIsStaleAndTheNextPeerGetsTheWholeWordPartition(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	wordPartition := fixtureWordPartitions(1)[0]
	urls := urlsOfPartition(2)
	excludedHolder := peerAfter(wordPartition, 0)
	holder, nextPeer := peerAfter(wordPartition, 1), peerAfter(wordPartition, 2)
	h.admit(t, wordPartition, urls[0])
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, excludedHolder.Hash, holder.Hash)
	h.admit(t, wordPartition, urls[1])
	h.eligibility.excludedPeers[excludedHolder.Hash] = true

	offer := onlyOffer(t, h.undeliveredBatch(t, excludedHolder, holder, nextPeer).Offers)

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
	assertOnlyStaleHolder(t, offer, excludedHolder)
	responsiblePeers := offer.OutcomeFrom(nil).ResponsiblePeers
	if !slices.Equal(responsiblePeers, []yacymodel.Hash{holder.Hash, nextPeer.Hash}) {
		t.Fatalf("ResponsiblePeers = %v, want the holder and the next peer", responsiblePeers)
	}
}

func TestAnUndeliveredWordPartitionWithARecipientInCooldownWaits(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	peer := peerAfter(wordPartition, 0)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.eligibility.peersInCooldown[peer.Hash] = true

	batch := h.undeliveredBatch(t, peer)

	assertWaits(t, batch.Waits,
		wordpartitionoffer.WaitRecipientInCooldown,
	)
}

func TestAnUndeliveredWordPartitionInRetryWaits(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.recordNoHolders(t, wordPartition)

	batch := h.undeliveredBatch(t, peerAfter(wordPartition, 0))

	assertWaits(t, batch.Waits,
		wordpartitionoffer.WaitWordPartitionInRetry,
	)
}

func TestAnUndeliveredWordPartitionWithoutRecipientsWaits(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	absentHolder := peerAfter(wordPartition, 1<<40)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.recordHolders(t, wordPartition, absentHolder.Hash)
	h.reachability.recentlyReachable[absentHolder.Hash] = true

	batch := h.undeliveredBatch(t, peerAfter(wordPartition, 0))

	assertWaits(
		t,
		batch.Waits,
		wordpartitionoffer.WaitNoRecipient,
	)
}

func TestADueRefreshWithARecipientInCooldownWaits(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	holder := peerAfter(wordPartition, 0)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)
	h.eligibility.peersInCooldown[holder.Hash] = true

	batch := h.refreshBatch(t, holder)

	assertWaits(
		t,
		batch.Waits,
		wordpartitionoffer.WaitRecipientInCooldown,
	)
}

func TestADueRefreshWithoutRecipientsWaits(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	absentHolder := peerAfter(wordPartition, 1<<40)
	h.admit(t, wordPartition, urlsOfPartition(1)...)
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, absentHolder.Hash)
	h.reachability.recentlyReachable[absentHolder.Hash] = true
	h.clock = h.clock.Add(refreshInterval)

	batch := h.refreshBatch(t, peerAfter(wordPartition, 0))

	assertWaits(
		t,
		batch.Waits,
		wordpartitionoffer.WaitNoRecipient,
	)
}

func TestAnOfferedWordPartitionDoesNotWait(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	holder := peerAfter(wordPartition, 0)
	h.admit(t, wordPartition, urlsOfPartition(2)...)
	h.recordHolders(t, wordPartition, holder.Hash)
	h.clock = h.clock.Add(refreshInterval)

	batch := h.refreshBatch(t, holder)

	if len(batch.Offers) == 0 || len(batch.Waits) != 0 {
		t.Fatalf("offers = %v and waits = %v, want an offer and no wait", batch.Offers, batch.Waits)
	}
}

func assertWaits(
	t *testing.T,
	waits []wordpartitionoffer.WaitReason,
	waitReason wordpartitionoffer.WaitReason,
) {
	t.Helper()

	if want := []wordpartitionoffer.WaitReason{waitReason}; !slices.Equal(waits, want) {
		t.Fatalf("waits = %v, want %v", waits, want)
	}
}
