package peerdirectory_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const wideCapacity = 16

type silentObserver struct{}

func (silentObserver) PeerAdmitted(context.Context, yacymodel.Hash, int)               {}
func (silentObserver) PeerAnswered(context.Context, yacymodel.Hash, string, time.Time) {}
func (silentObserver) PeerWentSilent(context.Context, yacymodel.Hash)                  {}
func (silentObserver) PeerDropped(context.Context, yacymodel.Hash)                     {}
func (silentObserver) PeersKnown(context.Context, int, int, int)                       {}

type heldPeersBeforeOfferedPeers struct{}

func (heldPeersBeforeOfferedPeers) StalestPeersFirst(
	_ context.Context,
	members []peerdirectory.KnownPeer,
	candidates []peerdirectory.CandidatePeer,
) []yacymodel.Hash {
	stalest := make([]yacymodel.Hash, 0, len(members)+len(candidates))
	for _, member := range slices.SortedFunc(slices.Values(members), oldestAdmittedFirst) {
		stalest = append(stalest, member.Hash)
	}
	for _, candidate := range candidates {
		stalest = append(stalest, candidate.Hash)
	}

	return stalest
}

func oldestAdmittedFirst(a, b peerdirectory.KnownPeer) int {
	return a.AdmittedAt.Compare(b.AdmittedAt)
}

type offeredPeersBeforeHeldPeers struct{}

func (offeredPeersBeforeHeldPeers) StalestPeersFirst(
	_ context.Context,
	members []peerdirectory.KnownPeer,
	candidates []peerdirectory.CandidatePeer,
) []yacymodel.Hash {
	stalest := make([]yacymodel.Hash, 0, len(members)+len(candidates))
	for _, candidate := range candidates {
		stalest = append(stalest, candidate.Hash)
	}
	for _, member := range members {
		stalest = append(stalest, member.Hash)
	}

	return stalest
}

type testClock struct{ instant time.Time }

func (c *testClock) now() time.Time { return c.instant }

func hashOf(t *testing.T, symbol byte) yacymodel.Hash {
	t.Helper()

	hash, err := yacymodel.ParseHash(string([]byte{
		symbol, symbol, symbol, symbol, symbol, symbol,
		symbol, symbol, symbol, symbol, symbol, symbol,
	}))
	if err != nil {
		t.Fatalf("parse hash: %v", err)
	}

	return hash
}

func seedOf(t *testing.T, hash yacymodel.Hash, host string) yacymodel.Seed {
	t.Helper()

	address, err := yacymodel.ParseHost(host)
	if err != nil {
		t.Fatalf("parse host %q: %v", host, err)
	}
	port, err := yacymodel.ParsePort("8090")
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	return yacymodel.Seed{
		Hash:           hash,
		PrimaryAddress: yacymodel.Some(address),
		Port:           yacymodel.Some(port),
	}
}

func directoryAt(clock *testClock, capacity int) *peerdirectory.Directory {
	return directoryOver(
		clock,
		peerdirectory.DirectoryLimits{Capacity: capacity},
		heldPeersBeforeOfferedPeers{},
		silentObserver{},
	)
}

func directoryOver(
	clock *testClock,
	limits peerdirectory.DirectoryLimits,
	stale peerdirectory.StalePeerSource,
	observer peerdirectory.DirectoryObserver,
) *peerdirectory.Directory {
	return peerdirectory.New(
		limits,
		clock.now,
		stale,
		peerdirectory.DirectoryObservers{observer},
	)
}

func TestAnAdmittedPeerIsNotAskedBeforeAnAddressAnswered(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, hashOf(t, 'a'), "10.0.0.1")})

	if askable := directory.AskablePeers(t.Context()); len(askable) != 0 {
		t.Fatalf("AskablePeers = %v, want none before a probe answered", askable)
	}
}

func TestAPeerBecomesAskableOnTheAddressThatAnswered(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")

	askable := directory.AskablePeers(t.Context())
	if len(askable) != 1 || askable[0].Address != "http://10.0.0.1:8090" {
		t.Fatalf("AskablePeers = %v, want the address that answered", askable)
	}
}

func TestASilentPeerLeavesTheAskableSet(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")
	directory.ConfirmSilent(t.Context(), peer)

	if askable := directory.AskablePeers(t.Context()); len(askable) != 0 {
		t.Fatalf("AskablePeers = %v, want none once the peer went silent", askable)
	}
}

func TestAdmittingAPeerThatNeverAnsweredReplacesTheAddressesItAdvertises(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.2")})

	known := directory.KnownPeers(t.Context())
	if len(known) != 1 || known[0].Addresses[0] != "http://10.0.0.2:8090" {
		t.Fatalf("KnownPeers = %+v, want one peer on 10.0.0.2", known)
	}
}

func TestAPeerThatAnswersAgainComesBackToTheAskableSet(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")
	clock.instant = clock.instant.Add(time.Second)
	directory.ConfirmSilent(t.Context(), peer)

	clock.instant = clock.instant.Add(time.Second)
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")

	if askable := directory.AskablePeers(t.Context()); len(askable) != 1 {
		t.Fatalf("AskablePeers = %v, want the peer that answered after its silence", askable)
	}
}

func TestASeedThatLeavesOutTheAddressAPeerAnsweredOnDoesNotTakeItAway(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")
	clock.instant = clock.instant.Add(time.Second)
	directory.ConfirmSilent(t.Context(), peer)

	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.2")})

	known := directory.KnownPeers(t.Context())
	if len(known) != 1 || !slices.Equal(known[0].Addresses, []string{
		"http://10.0.0.1:8090",
		"http://10.0.0.2:8090",
	}) {
		t.Fatalf("KnownPeers = %+v, want the answered address kept and leading", known)
	}
}

func TestASeedWithoutAPortIsNotAPeerTheDirectoryKnows(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, wideCapacity)
	directory.Admit(t.Context(), []yacymodel.Seed{{Hash: hashOf(t, 'a')}})

	if known := directory.KnownPeers(t.Context()); len(known) != 0 {
		t.Fatalf("KnownPeers = %+v, want none", known)
	}
}

func TestAFullDirectoryDropsTheStalestPeerToAdmitANewOne(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryAt(clock, 1)
	first, second := hashOf(t, 'a'), hashOf(t, 'b')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, first, "10.0.0.1")})
	clock.instant = clock.instant.Add(time.Minute)
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, second, "10.0.0.2")})

	known := directory.KnownPeers(t.Context())
	if len(known) != 1 || known[0].Hash != second {
		t.Fatalf("KnownPeers = %+v, want only the newly admitted peer", known)
	}
}

type contentsRecorder struct {
	silentObserver
	peers          int
	answeringPeers int
}

func (r *contentsRecorder) PeersKnown(_ context.Context, peers, answeringPeers, _ int) {
	r.peers = peers
	r.answeringPeers = answeringPeers
}

func TestTheDirectoryReportsHowManyOfItsPeersAnswer(t *testing.T) {
	t.Parallel()

	recorder := &contentsRecorder{}
	directory := directoryOver(
		&testClock{instant: time.Unix(0, 0)},
		peerdirectory.DirectoryLimits{Capacity: wideCapacity},
		heldPeersBeforeOfferedPeers{},
		recorder,
	)
	answering, silent := hashOf(t, 'a'), hashOf(t, 'b')
	directory.Admit(t.Context(), []yacymodel.Seed{
		seedOf(t, answering, "10.0.0.1"),
		seedOf(t, silent, "10.0.0.2"),
	})
	directory.ConfirmAnswering(t.Context(), answering, "http://10.0.0.1:8090")
	directory.ConfirmSilent(t.Context(), silent)

	if recorder.peers != 2 || recorder.answeringPeers != 1 {
		t.Fatalf("PeersKnown = %d peers, %d answering, want 2 and 1",
			recorder.peers, recorder.answeringPeers)
	}
}

type reportedAnswer struct {
	peer       yacymodel.Hash
	address    string
	answeredAt time.Time
}

type answerReportingObserver struct {
	silentObserver
	answers         []reportedAnswer
	droppedPeers    []yacymodel.Hash
	wentSilentPeers []yacymodel.Hash
}

func (o *answerReportingObserver) PeerAnswered(
	_ context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	o.answers = append(
		o.answers,
		reportedAnswer{peer: peer, address: address, answeredAt: answeredAt},
	)
}

func (o *answerReportingObserver) PeerDropped(_ context.Context, peer yacymodel.Hash) {
	o.droppedPeers = append(o.droppedPeers, peer)
}

func (o *answerReportingObserver) PeerWentSilent(_ context.Context, peer yacymodel.Hash) {
	o.wentSilentPeers = append(o.wentSilentPeers, peer)
}

func TestAnAnsweringPeerIsReportedWithItsAddressAndTheTimeItAnswered(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	answers := &answerReportingObserver{}
	directory := directoryOver(
		clock,
		peerdirectory.DirectoryLimits{Capacity: wideCapacity},
		heldPeersBeforeOfferedPeers{},
		answers,
	)
	peer := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, peer, "10.0.0.1")})

	clock.instant = clock.instant.Add(time.Minute)
	directory.ConfirmAnswering(t.Context(), peer, "http://10.0.0.1:8090")

	want := reportedAnswer{peer: peer, address: "http://10.0.0.1:8090", answeredAt: clock.instant}
	if len(answers.answers) != 1 || answers.answers[0] != want {
		t.Fatalf("answers reported %+v, want %+v", answers.answers, want)
	}
}

func TestOnlyAPeerThatWasAnsweringIsReportedAsGoneSilent(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	answers := &answerReportingObserver{}
	directory := directoryOver(
		clock,
		peerdirectory.DirectoryLimits{Capacity: wideCapacity},
		heldPeersBeforeOfferedPeers{},
		answers,
	)
	answering, neverAnswering := hashOf(t, 'a'), hashOf(t, 'b')
	directory.Admit(t.Context(), []yacymodel.Seed{
		seedOf(t, answering, "10.0.0.1"),
		seedOf(t, neverAnswering, "10.0.0.2"),
	})
	directory.ConfirmAnswering(t.Context(), answering, "http://10.0.0.1:8090")

	clock.instant = clock.instant.Add(time.Minute)
	directory.ConfirmSilent(t.Context(), answering)
	directory.ConfirmSilent(t.Context(), neverAnswering)
	clock.instant = clock.instant.Add(time.Minute)
	directory.ConfirmSilent(t.Context(), answering)

	if !slices.Equal(answers.wentSilentPeers, []yacymodel.Hash{answering}) {
		t.Fatalf(
			"peers reported gone silent %v, want only %v once",
			answers.wentSilentPeers,
			answering,
		)
	}
}

func TestAPeerEvictedToMakeRoomIsReportedAsDropped(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	answers := &answerReportingObserver{}
	directory := directoryOver(
		clock,
		peerdirectory.DirectoryLimits{Capacity: 1},
		heldPeersBeforeOfferedPeers{},
		answers,
	)
	evicted := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, evicted, "10.0.0.1")})

	clock.instant = clock.instant.Add(time.Minute)
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, hashOf(t, 'b'), "10.0.0.2")})

	if len(answers.droppedPeers) != 1 || answers.droppedPeers[0] != evicted {
		t.Fatalf("dropped peers reported %v, want %v", answers.droppedPeers, evicted)
	}
}

func seedsOf(t *testing.T, symbols string) []yacymodel.Seed {
	t.Helper()

	seeds := make([]yacymodel.Seed, 0, len(symbols))
	for _, symbol := range []byte(symbols) {
		seeds = append(seeds, seedOf(t, hashOf(t, symbol), "10.0.0.1"))
	}

	return seeds
}

func heldHashes(t *testing.T, directory *peerdirectory.Directory) map[yacymodel.Hash]struct{} {
	t.Helper()

	held := map[yacymodel.Hash]struct{}{}
	for _, peer := range directory.KnownPeers(t.Context()) {
		held[peer.Hash] = struct{}{}
	}

	return held
}

func TestAFullDirectoryRefusesACandidateThatIsStalerThanEveryPeerItHolds(t *testing.T) {
	t.Parallel()

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryOver(
		clock,
		peerdirectory.DirectoryLimits{Capacity: 1},
		offeredPeersBeforeHeldPeers{},
		silentObserver{},
	)
	held := hashOf(t, 'a')
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, held, "10.0.0.1")})
	directory.Admit(t.Context(), []yacymodel.Seed{seedOf(t, hashOf(t, 'b'), "10.0.0.2")})

	known := directory.KnownPeers(t.Context())
	if len(known) != 1 || known[0].Hash != held {
		t.Fatalf("KnownPeers = %+v, want the peer the directory already held", known)
	}
}

func TestAFullDirectoryLendsItsNewcomerShareToPeersTheOrderRefuses(t *testing.T) {
	t.Parallel()

	const (
		membersOfAFullDirectory = 16
		newcomerShare           = 0.25
		newcomersLentASlot      = 4
	)

	clock := &testClock{instant: time.Unix(0, 0)}
	directory := directoryOver(
		clock,
		peerdirectory.DirectoryLimits{
			Capacity:      membersOfAFullDirectory,
			NewcomerShare: newcomerShare,
		},
		offeredPeersBeforeHeldPeers{},
		silentObserver{},
	)
	directory.Admit(t.Context(), seedsOf(t, "abcdefghijklmnop"))
	directory.Admit(t.Context(), seedsOf(t, "ABCDEFGHIJKLMNOP"))

	held := heldHashes(t, directory)
	newcomersHeld := 0
	for _, seed := range seedsOf(t, "ABCDEFGHIJKLMNOP") {
		if _, isHeld := held[seed.Hash]; isHeld {
			newcomersHeld++
		}
	}
	if len(held) != membersOfAFullDirectory || newcomersHeld != newcomersLentASlot {
		t.Fatalf(
			"the directory holds %d peers, %d of them newcomers, want %d and %d",
			len(held),
			newcomersHeld,
			membersOfAFullDirectory,
			newcomersLentASlot,
		)
	}
}

func TestTheNewcomersLentASlotAreDrawnFromTheWholeAdmission(t *testing.T) {
	t.Parallel()

	const (
		membersOfAFullDirectory = 16
		newcomerShare           = 0.25
		admissions              = 10
	)

	reachedTheEndOfTheAdmission := false
	for range admissions {
		clock := &testClock{instant: time.Unix(0, 0)}
		directory := directoryOver(
			clock,
			peerdirectory.DirectoryLimits{
				Capacity:      membersOfAFullDirectory,
				NewcomerShare: newcomerShare,
			},
			offeredPeersBeforeHeldPeers{},
			silentObserver{},
		)
		directory.Admit(t.Context(), seedsOf(t, "abcdefghijklmnop"))
		directory.Admit(t.Context(), seedsOf(t, "ABCDEFGHIJKLMNOP"))

		held := heldHashes(t, directory)
		for _, seed := range seedsOf(t, "MNOP") {
			if _, isHeld := held[seed.Hash]; isHeld {
				reachedTheEndOfTheAdmission = true
			}
		}
	}
	if !reachedTheEndOfTheAdmission {
		t.Fatalf(
			"%d admissions lent no slot to a newcomer named last, want the whole admission drawn from",
			admissions,
		)
	}
}
