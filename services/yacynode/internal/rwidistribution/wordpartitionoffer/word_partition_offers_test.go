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
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
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
	writer       rwipostings.PostingWriter
	undelivered  *undeliveredpostings.Postings
	schedule     *wordpartitionschedule.Schedule
	offers       *wordpartitionoffer.Offers
	reachability *fakeReachability
	eligibility  fakeEligibility
	clock        time.Time
	admitted     []yacymodel.WordPartition
}

func openHarness(t *testing.T, opts options) *harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil, wallclock.Clock{})
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
	index, writer, _, err := rwipostings.Open(
		context.Background(),
		v,
		partitions(),
		nil,
		nil,
		[]vault.Projection[yacymodel.RWIPosting]{projection},
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	h := &harness{
		vault:        v,
		undelivered:  undelivered,
		reachability: &fakeReachability{recentlyReachable: map[yacymodel.Hash]bool{}},
		eligibility: fakeEligibility{
			pausedPeers:   map[yacymodel.Hash]bool{},
			excludedPeers: map[yacymodel.Hash]bool{},
			busyPeers:     map[yacymodel.Hash]bool{},
		},
		clock: time.Unix(1_700_000_000, 0).UTC(),
	}
	h.schedule, err = wordpartitionschedule.Open(
		context.Background(),
		v,
		partitions(),
		refreshInterval,
		retryDelay,
		h.now,
	)
	if err != nil {
		t.Fatalf("wordpartitionschedule.Open: %v", err)
	}
	h.writer = h.schedule.WriterFrom(writer, silentAdmissionObserver{})
	h.offers = wordpartitionoffer.New(
		v, undelivered, h.schedule, index, h.reachability, h.eligibility, configFrom(opts),
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

	if !slices.Contains(h.admitted, wordPartition) {
		h.admitted = append(h.admitted, wordPartition)
	}
	h.write(t, func(tx *vault.Txn) error {
		for _, url := range urls {
			posting := yacymodel.RWIPosting{
				WordHash: wordPartition.Word,
				URLHash:  url,
				Language: yacymodel.LanguageOfUndeclaredDocument,
			}
			if err := h.writer.Write(tx, posting); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h *harness) deliverEverything(t *testing.T) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, wordPartition := range h.admitted {
			undelivered, err := h.undelivered.PostingsOf(tx, wordPartition)
			if err != nil {
				return err
			}
			for _, posting := range undelivered {
				if _, err := h.undelivered.MarkDelivered(tx, posting); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

var wholeRing = yacymodel.DHTRingSpan{First: 0, Last: yacymodel.MaxDHTRingPosition}

func (h *harness) dueWordPartitions(t *testing.T) []yacymodel.WordPartition {
	t.Helper()

	var dueWordPartitions []yacymodel.WordPartition
	h.read(t, func(tx *vault.Txn) error {
		dueRefreshes, _, err := h.schedule.WordPartitionsDueWithin(
			tx, wholeRing, wordpartitionschedule.ScheduledWordPartition{}, everyEntryBudget)
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
		return h.schedule.RecordRefreshed(tx, wordPartition, recordedHolders)
	})
}

func (h *harness) recordNoHolders(t *testing.T, wordPartition yacymodel.WordPartition) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		return h.schedule.RecordShortOfRedundancy(tx, wordPartition, nil)
	})
}

func (h *harness) dueOffers(
	t *testing.T,
	reachablePeers ...yacymodel.Seed,
) []wordpartitionoffer.Offer {
	t.Helper()

	offers, _ := h.dueOffersWithin(t, everyEntryBudget, reachablePeers...)

	return offers
}

func (h *harness) dueOffersWithin(
	t *testing.T,
	amountOfPostingsPerPeer int,
	reachablePeers ...yacymodel.Seed,
) ([]wordpartitionoffer.Offer, bool) {
	t.Helper()

	h.reachability.reachable = reachablePeers
	offers, isBudgetSpent, err := h.offers.DueOffersWithin(
		context.Background(), h.dueWordPartitions(t), reachablePeers, amountOfPostingsPerPeer)
	if err != nil {
		t.Fatalf("DueOffersWithin: %v", err)
	}

	return offers, isBudgetSpent
}

func (h *harness) scheduleAdmissions(t *testing.T) {
	t.Helper()

	err := h.schedule.ScheduleAdmissionsWithin(
		context.Background(),
		[]yacymodel.DHTRingSpan{wholeRing},
	)
	if err != nil {
		t.Fatalf("ScheduleAdmissionsWithin: %v", err)
	}
}

type silentAdmissionObserver struct{}

func (silentAdmissionObserver) ObservePostingAdmitted(yacymodel.DHTRingPosition) {}

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
		Hash:         yacymodel.HashFromDHTRingPosition(position),
		PrimaryHost:  yacymodel.Some(host),
		Port:         yacymodel.Some(port),
		Capabilities: yacymodel.Some(yacymodel.PeerCapabilities{AcceptRemoteIndex: true}),
	}
}

func onlyOffer(
	t *testing.T,
	offers []wordpartitionoffer.Offer,
) wordpartitionoffer.Offer {
	t.Helper()

	if len(offers) != 1 {
		t.Fatalf("offers = %+v, want one", offers)
	}

	return offers[0]
}

func wordPartitionsOf(offers []wordpartitionoffer.Offer) []yacymodel.WordPartition {
	wordPartitions := make([]yacymodel.WordPartition, 0, len(offers))
	for _, offer := range offers {
		wordPartitions = append(wordPartitions, offer.WordPartition())
	}

	return wordPartitions
}

func recipientPeersOf(offer wordpartitionoffer.Offer) []yacymodel.Hash {
	peers := make([]yacymodel.Hash, 0, len(offer.Recipients))
	for _, recipient := range offer.Recipients {
		peers = append(peers, recipient.Peer.Hash)
	}

	return peers
}

func postingsSentTo(
	t *testing.T,
	offer wordpartitionoffer.Offer,
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
	offer wordpartitionoffer.Offer,
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

func undeliveredURLsOf(offer wordpartitionoffer.Offer) []yacymodel.URLHash {
	return urlsOfUndelivered(offer.OutcomeFrom(everythingAcceptedIn(offer)).DeliveredPostings)
}

func everythingAcceptedIn(
	offer wordpartitionoffer.Offer,
) map[yacymodel.Hash]wordpartitionoffer.PeerAnswer {
	answers := make(map[yacymodel.Hash]wordpartitionoffer.PeerAnswer, len(offer.Recipients))
	for _, recipient := range offer.Recipients {
		answers[recipient.Peer.Hash] = accepted(recipient.Postings)
	}

	return answers
}

func urlsOfUndelivered(postings []undeliveredpostings.Posting) []yacymodel.URLHash {
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
	offer wordpartitionoffer.Offer,
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
	pausedPeers   map[yacymodel.Hash]bool
	excludedPeers map[yacymodel.Hash]bool
	busyPeers     map[yacymodel.Hash]bool
}

func (f fakeEligibility) IsPaused(peer yacymodel.Hash) bool {
	return f.pausedPeers[peer]
}

func (f fakeEligibility) IsExcluded(peer yacymodel.Hash) bool {
	return f.excludedPeers[peer]
}

func (f fakeEligibility) IsBusy(peer yacymodel.Hash) bool {
	return f.busyPeers[peer]
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

	h.scheduleAdmissions(t)
	offers := h.dueOffers(t, holder, missingHolder)

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
		t.Fatalf("undelivered postings = %v, want %v", got, want)
	}
}

func TestAWordPartitionWithARecipientPausedOrBusyWaitsWhileTheOthersGoOut(t *testing.T) {
	for name, holdBack := range map[string]func(h *harness, peer yacymodel.Hash){
		"paused": func(h *harness, peer yacymodel.Hash) { h.eligibility.pausedPeers[peer] = true },
		"busy":   func(h *harness, peer yacymodel.Hash) { h.eligibility.busyPeers[peer] = true },
	} {
		t.Run(name, func(t *testing.T) {
			h := openHarness(t, options{})
			wordPartitions := fixtureWordPartitions(2)
			waiting, free := wordPartitions[0], wordPartitions[1]
			heldBackPeer, freePeer := peerAfter(waiting, 0), peerAfter(free, 0)
			h.admit(t, waiting, urlsOfPartition(1)...)
			h.admit(t, free, urlsOfPartition(1)...)
			holdBack(h, heldBackPeer.Hash)

			h.scheduleAdmissions(t)
			offers := h.dueOffers(t, heldBackPeer, freePeer)

			want := []yacymodel.WordPartition{free}
			if got := wordPartitionsOf(offers); !slices.Equal(got, want) {
				t.Fatalf("offers = %v, want only %v while %v waits", got, free, waiting)
			}
		})
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

	refresh := onlyOffer(t, h.dueOffers(t, holder, missingHolder))

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

	refresh := onlyOffer(t, h.dueOffers(t, holder))

	if outcome := refresh.OutcomeFrom(nil); outcome.WordPartition != refreshed ||
		!outcome.IsRefresh {
		t.Fatalf("offer = %+v, want the refresh of %v", refresh, refreshed)
	}
	if got, want := undeliveredURLsOf(refresh), urls[2:]; !slices.Equal(got, want) {
		t.Fatalf("undelivered postings = %v, want %v", got, want)
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

	refresh := onlyOffer(t, h.dueOffers(t, responsiblePeers...))

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

	h.scheduleAdmissions(t)
	offer := onlyOffer(t, h.dueOffers(t, closest, farther))

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

	h.scheduleAdmissions(t)
	offers := h.dueOffers(t, responsiblePeer, otherPeer)

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

			h.scheduleAdmissions(t)
			offer := onlyOffer(t, h.dueOffers(t, reachablePeers...))

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

	h.scheduleAdmissions(t)
	offer := onlyOffer(t, h.dueOffers(t, refusingPeer, acceptingPeer))

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
	h.scheduleAdmissions(t)

	if offers := h.dueOffers(t, peer); len(offers) != 0 {
		t.Fatalf("offers before the retry = %v, want none", offers)
	}

	h.clock = h.clock.Add(retryDelay)
	refresh := onlyOffer(t, h.dueOffers(t, peer))

	if !refresh.OutcomeFrom(nil).IsRefresh {
		t.Fatalf("offer at the retry = %+v, want a refresh", refresh)
	}
	if got := undeliveredURLsOf(refresh); !slices.Equal(got, urls) {
		t.Fatalf("undelivered postings = %v, want %v", got, urls)
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

	h.scheduleAdmissions(t)
	offer := onlyOffer(t, h.dueOffers(t, excludedHolder, holder, nextPeer))

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

func TestADueWordPartitionWithoutUndeliveredPostingsGetsARefreshOffer(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	holder := peerAfter(wordPartition, 0)
	urls := urlsOfPartition(2)
	h.admit(t, wordPartition, urls...)
	h.deliverEverything(t)
	h.recordHolders(t, wordPartition, holder.Hash)
	h.scheduleAdmissions(t)

	offer := onlyOffer(t, h.dueOffers(t, holder))

	if !offer.OutcomeFrom(nil).IsRefresh {
		t.Fatalf("offer = %+v, want a refresh", offer)
	}
	if got := urlsSentTo(t, offer, holder); !slices.Equal(got, sortedURLs(urls)) {
		t.Fatalf("holder gets %v, want the whole word partition %v", got, urls)
	}
}

func TestANeverRefreshedWordPartitionGetsTheWholeWordPartitionSentToEveryRecipient(t *testing.T) {
	h := openHarness(t, options{redundancy: 2})
	wordPartition := fixtureWordPartitions(1)[0]
	urls := urlsOfPartition(2)
	peers := []yacymodel.Seed{peerAfter(wordPartition, 0), peerAfter(wordPartition, 1)}
	h.admit(t, wordPartition, urls...)
	h.scheduleAdmissions(t)

	offer := onlyOffer(t, h.dueOffers(t, peers...))

	if !offer.OutcomeFrom(nil).IsRefresh {
		t.Fatalf("offer = %+v, want a refresh", offer)
	}
	for _, peer := range peers {
		if got := urlsSentTo(t, offer, peer); !slices.Equal(got, sortedURLs(urls)) {
			t.Fatalf("%v gets %v, want the whole word partition %v", peer.Hash, got, urls)
		}
	}
}

func TestArcsAmongLeaveOutPeersThatRefuseRemoteIndexOrAreExcluded(t *testing.T) {
	h := openHarness(t, options{redundancy: 1})
	wordPartition := fixtureWordPartitions(1)[0]
	accepting, refusing, excluded := peerAfter(wordPartition, 0), peerAfter(wordPartition, 1),
		peerAfter(wordPartition, 2)
	refusing.Capabilities = yacymodel.Some(yacymodel.PeerCapabilities{})
	h.eligibility.excludedPeers[excluded.Hash] = true

	arcs := h.offers.ArcsAmong([]yacymodel.Seed{accepting, refusing, excluded})

	if len(arcs) != 1 || len(arcs[0].Window) != 1 || arcs[0].Window[0].Hash != accepting.Hash {
		t.Fatalf("arcs = %+v, want one arc with only the accepting peer", arcs)
	}
}

func TestOffersStopBeforeOneThatWouldGiveAPeerMoreThanItsBudget(t *testing.T) {
	h := openHarness(t, options{})
	wordPartitions := fixtureWordPartitions(2)
	peer := peerAfter(wordPartitions[1], 0)
	for _, wordPartition := range wordPartitions {
		h.admit(t, wordPartition, urlsOfPartition(2)...)
	}
	h.scheduleAdmissions(t)

	offers, isBudgetSpent := h.dueOffersWithin(t, 3, peer)

	if got, want := wordPartitionsOf(offers), h.dueWordPartitions(t)[:1]; !slices.Equal(got, want) {
		t.Fatalf("offers = %v, want only the first due word partition %v", got, want)
	}
	if !isBudgetSpent {
		t.Fatalf("budget spent = false, want true with a word partition left")
	}
}

func TestOffersKeepTheFirstOneEvenWhenItAloneIsOverTheBudget(t *testing.T) {
	h := openHarness(t, options{})
	wordPartition := fixtureWordPartitions(1)[0]
	h.admit(t, wordPartition, urlsOfPartition(2)...)
	h.scheduleAdmissions(t)

	offers, isBudgetSpent := h.dueOffersWithin(t, 1, peerAfter(wordPartition, 0))

	if got := wordPartitionsOf(
		offers,
	); !slices.Equal(
		got,
		[]yacymodel.WordPartition{wordPartition},
	) {
		t.Fatalf("offers = %v, want %v", got, wordPartition)
	}
	if isBudgetSpent {
		t.Fatalf("budget spent = true, want false with nothing left")
	}
}
