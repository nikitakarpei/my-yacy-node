package urlmetadataasks_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/documentholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	askCeiling      = 20
	asksPerDocument = 2
	grace           = time.Minute
)

var cutoffAtNinetyPercent = urlmetadataasks.Cutoff{PercentOfDocuments: 90, Grace: grace}

type peersOfTheNetwork struct {
	stuckPeers          map[string]struct{}
	failingPeers        map[string]struct{}
	latePeers           map[string]struct{}
	lateAnswersReleased <-chan struct{}
	asks                []peerasks.URLMetadataAsk
	asksContext         context.Context
}

func peersWhere(stuckPeers []string, failingPeers []string) *peersOfTheNetwork {
	peers := &peersOfTheNetwork{
		stuckPeers:   map[string]struct{}{},
		failingPeers: map[string]struct{}{},
		latePeers:    map[string]struct{}{},
	}
	for _, address := range stuckPeers {
		peers.stuckPeers[address] = struct{}{}
	}
	for _, address := range failingPeers {
		peers.failingPeers[address] = struct{}{}
	}

	return peers
}

func (peers *peersOfTheNetwork) AskForURLMetadata(
	ctx context.Context,
	asks []peerasks.URLMetadataAsk,
) <-chan peerasks.URLMetadataAskOutcome {
	peers.asks = append(peers.asks, asks...)
	peers.asksContext = ctx
	outcomesAsTheySettle := make(chan peerasks.URLMetadataAskOutcome, len(asks))
	anyStuck := false
	lateAsks := make([]peerasks.URLMetadataAsk, 0, len(asks))
	for _, ask := range asks {
		if _, stuck := peers.stuckPeers[ask.Peer.Address]; stuck {
			anyStuck = true

			continue
		}
		if _, late := peers.latePeers[ask.Peer.Address]; late {
			lateAsks = append(lateAsks, ask)

			continue
		}
		outcomesAsTheySettle <- peers.outcomeOf(ask)
	}
	if !anyStuck && len(lateAsks) == 0 {
		close(outcomesAsTheySettle)

		return outcomesAsTheySettle
	}
	go func() {
		peers.answerLate(ctx, lateAsks, outcomesAsTheySettle)
		if anyStuck {
			<-ctx.Done()
		}
		close(outcomesAsTheySettle)
	}()

	return outcomesAsTheySettle
}

func (peers *peersOfTheNetwork) answerLate(
	ctx context.Context,
	lateAsks []peerasks.URLMetadataAsk,
	outcomesAsTheySettle chan<- peerasks.URLMetadataAskOutcome,
) {
	if len(lateAsks) == 0 {
		return
	}
	select {
	case <-peers.lateAnswersReleased:
	case <-ctx.Done():
		return
	}
	for _, ask := range lateAsks {
		outcomesAsTheySettle <- peers.outcomeOf(ask)
	}
}

func (peers *peersOfTheNetwork) outcomeOf(
	ask peerasks.URLMetadataAsk,
) peerasks.URLMetadataAskOutcome {
	outcome := peerasks.URLMetadataAskOutcome{Ask: ask, Put: true}
	if _, fails := peers.failingPeers[ask.Peer.Address]; fails {
		return outcome
	}
	metadataOfEachDocument := make([]yacymodel.URLMetadata, 0, len(ask.Documents))
	for _, document := range ask.Documents {
		metadataOfEachDocument = append(
			metadataOfEachDocument,
			yacymodel.URLMetadata{Hash: document},
		)
	}
	outcome.Answer = yacymodel.Some(peerasks.AnsweredURLMetadataAsk{
		Ask:                    ask,
		MetadataOfEachDocument: metadataOfEachDocument,
	})

	return outcome
}

type ceilingsOfThePeers struct {
	ceilingOfEachPeer map[string]int
}

func (ceilings ceilingsOfThePeers) CeilingOf(_ context.Context, address string) int {
	if ceiling, lowered := ceilings.ceilingOfEachPeer[address]; lowered {
		return ceiling
	}

	return askCeiling
}

type graceClock struct {
	firesAtOnce   bool
	gracesStarted int
	graceStarts   func()
}

func (clock *graceClock) After(_ time.Duration, expire func()) func() {
	clock.gracesStarted++
	if clock.graceStarts != nil {
		clock.graceStarts()
	}
	if clock.firesAtOnce {
		expire()
	}

	return func() {}
}

type peerAbstract struct {
	address   string
	documents []yacymodel.URLHash
}

func answersOf(abstracts ...peerAbstract) []wordpartitionasks.ReplicaAnswer {
	answers := make([]wordpartitionasks.ReplicaAnswer, 0, len(abstracts))
	for _, abstract := range abstracts {
		answer := wordpartitionasks.ReplicaAnswer{
			Replica: peerdirectory.AskablePeer{
				Hash:    yacymodel.WordHash(abstract.address),
				Address: abstract.address,
			},
		}
		for _, document := range abstract.documents {
			answer.ListedDocuments = append(
				answer.ListedDocuments, wordpartitionasks.ListedDocument{Hash: document},
			)
		}
		answers = append(answers, answer)
	}

	return answers
}

func documentsOf(t *testing.T, site string, amountOfDocuments int) []yacymodel.URLHash {
	t.Helper()

	documents := make([]yacymodel.URLHash, 0, amountOfDocuments)
	for number := range amountOfDocuments {
		document, err := yacymodel.URLHashOf(fmt.Sprintf("https://%s.example/%d", site, number))
		if err != nil {
			t.Fatalf("document %d of %q has no hash: %v", number, site, err)
		}
		documents = append(documents, document)
	}

	return documents
}

type lookupReports struct {
	performed []urlmetadataasks.Performed
}

func (reports *lookupReports) URLMetadataLookupPerformed(
	_ context.Context,
	performed urlmetadataasks.Performed,
) {
	reports.performed = append(reports.performed, performed)
}

const amountOfArrivalsAFakeHolds = 64

type metadataThePeersSent struct {
	metadataPerPeer map[yacymodel.Hash][]yacymodel.URLMetadata
	arrivals        chan struct{}
}

func (sent *metadataThePeersSent) PeerSentURLMetadata(
	peer yacymodel.Hash,
	metadataOfEachDocument []yacymodel.URLMetadata,
) {
	sent.metadataPerPeer[peer] = append(sent.metadataPerPeer[peer], metadataOfEachDocument...)
	sent.arrivals <- struct{}{}
}

type askedPeers struct {
	peers           *peersOfTheNetwork
	ceilings        ceilingsOfThePeers
	cutoff          urlmetadataasks.Cutoff
	clock           *graceClock
	asksPerDocument int
	reports         *lookupReports
	sentMetadata    *metadataThePeersSent
}

func askedPeersOf(peers *peersOfTheNetwork) askedPeers {
	return askedPeers{
		peers:           peers,
		clock:           &graceClock{},
		asksPerDocument: asksPerDocument,
		reports:         &lookupReports{},
		sentMetadata: &metadataThePeersSent{
			metadataPerPeer: map[yacymodel.Hash][]yacymodel.URLMetadata{},
			arrivals:        make(chan struct{}, amountOfArrivalsAFakeHolds),
		},
	}
}

func (asked askedPeers) lookupBegunIn(ctx context.Context) *urlmetadataasks.Lookup {
	return urlmetadataasks.New(
		asked.peers,
		asked.ceilings,
		asked.cutoff,
		asked.clock,
		asked.asksPerDocument,
		asked.reports,
	).Begin(ctx, asked.sentMetadata)
}

func (asked askedPeers) settledFor(
	ctx context.Context,
	answersOfEachJoin ...[]wordpartitionasks.ReplicaAnswer,
) urlmetadataasks.Performed {
	lookup := asked.lookupBegunIn(ctx)
	for _, answers := range answersOfEachJoin {
		lookup.AskFor(holdersIn(answers))
	}
	lookup.End()

	return asked.reports.performed[0]
}

func holdersIn(answers []wordpartitionasks.ReplicaAnswer) documentholders.Holders {
	holders := documentholders.NoneYet()
	holders.WordPartitionAnswered(yacymodel.WordHash("berlin"), 0, answers)

	return holders.HoldersOf(documentsAcross(answers))
}

func documentsAcross(answers []wordpartitionasks.ReplicaAnswer) yacymodel.URLHashes {
	documents := yacymodel.URLHashes{}
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			documents.Add(listedDocument.Hash)
		}
	}

	return documents
}

func TestTheAsksEndOnceTheirAnswersCoverEveryDocumentWithoutTheStuckPeer(t *testing.T) {
	t.Parallel()

	joined := documentsOf(t, "joined", 1)
	asked := askedPeersOf(peersWhere([]string{"second"}, nil))

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: joined},
		peerAbstract{address: "second", documents: joined},
	))

	if performed.EndReason != urlmetadataasks.EndedByCoverage ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 1 {
		t.Fatalf("the asks reported %+v, want them to end by coverage with the document", performed)
	}
}

func TestTheAsksWaitForTheSlowPeerWhoseDocumentNoOtherPeerSent(t *testing.T) {
	t.Parallel()

	shared := documentsOf(t, "shared", 1)
	onlySlow := documentsOf(t, "only-slow", 1)
	asked := askedPeersOf(peersWhere(nil, []string{"failing"}))

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: shared},
		peerAbstract{address: "failing", documents: documentsOf(t, "only-failing", 1)},
		peerAbstract{address: "slow", documents: append(shared, onlySlow...)},
	))

	if performed.EndReason != urlmetadataasks.EndedByEveryAskSettled ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 2 {
		t.Fatalf(
			"the asks reported %+v, want them to end once every ask settled, "+
				"with the document only the slow peer sent",
			performed,
		)
	}
}

func TestTheAsksAreCutOffAGraceAfterMostDocumentsSettled(t *testing.T) {
	t.Parallel()

	peers := peersWhere([]string{"stuck"}, nil)
	asked := askedPeersOf(peers)
	asked.cutoff = cutoffAtNinetyPercent
	asked.clock.firesAtOnce = true

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "fast", documents: documentsOf(t, "fast", 9)},
		peerAbstract{address: "stuck", documents: documentsOf(t, "stuck", 1)},
	))

	if performed.EndReason != urlmetadataasks.EndedByCutoff ||
		performed.AmountOfDocumentsCutOff != 1 ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 9 {
		t.Fatalf(
			"the asks reported %+v, want them cut off with the document of the stuck peer cut off",
			performed,
		)
	}
	if peers.asksContext.Err() == nil {
		t.Fatal("the ask of the stuck peer is still in flight, want it cancelled")
	}
}

func TestTheAsksWithTheCutoffOffWaitUntilTheQueryEnds(t *testing.T) {
	t.Parallel()

	asked := askedPeersOf(peersWhere([]string{"stuck"}, nil))
	asked.cutoff = urlmetadataasks.Cutoff{Grace: grace}
	asked.clock.firesAtOnce = true
	endedQuery, endQuery := context.WithCancel(t.Context())
	endQuery()

	performed := asked.settledFor(endedQuery, answersOf(
		peerAbstract{address: "fast", documents: documentsOf(t, "fast", 9)},
		peerAbstract{address: "stuck", documents: documentsOf(t, "stuck", 1)},
	))

	if performed.EndReason != urlmetadataasks.EndedByEveryAskSettled ||
		performed.AmountOfDocumentsCutOff != 0 || asked.clock.gracesStarted != 0 {
		t.Fatalf(
			"the asks reported %+v after %d graces, want them to wait for the stuck peer "+
				"until the query ends",
			performed, asked.clock.gracesStarted,
		)
	}
}

func TestADocumentOnlyARefusingPeerHoldsCountsAsSettled(t *testing.T) {
	t.Parallel()

	asked := askedPeersOf(peersWhere([]string{"stuck"}, []string{"refusing"}))
	asked.cutoff = cutoffAtNinetyPercent
	asked.clock.firesAtOnce = true

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "fast", documents: documentsOf(t, "fast", 16)},
		peerAbstract{address: "refusing", documents: documentsOf(t, "refusing", 3)},
		peerAbstract{address: "stuck", documents: documentsOf(t, "stuck", 1)},
	))

	if performed.EndReason != urlmetadataasks.EndedByCutoff ||
		performed.AmountOfDocumentsCutOff != 1 ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 16 {
		t.Fatalf(
			"the asks reported %+v, want them cut off with the documents of the refusing "+
				"peer settled and only the document of the stuck peer cut off",
			performed,
		)
	}
}

func TestADocumentAnsweredByOnePeerIsSettledWhileAnotherPeerIsStuck(t *testing.T) {
	t.Parallel()

	fastDocuments := documentsOf(t, "fast", 9)
	asked := askedPeersOf(peersWhere([]string{"stuck"}, nil))
	asked.cutoff = cutoffAtNinetyPercent
	asked.clock.firesAtOnce = true

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "fast", documents: fastDocuments},
		peerAbstract{
			address:   "stuck",
			documents: append(fastDocuments[:1:1], documentsOf(t, "only-stuck", 1)...),
		},
	))

	if performed.EndReason != urlmetadataasks.EndedByCutoff ||
		performed.AmountOfDocumentsCutOff != 1 ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 9 {
		t.Fatalf(
			"the asks reported %+v, want the document the fast peer sent settled and "+
				"only the document of the stuck peer cut off",
			performed,
		)
	}
}

func TestTheAsksEndByCoverageWhenTheLastDocumentComesBeforeTheGraceEnds(t *testing.T) {
	t.Parallel()

	fastDocuments := documentsOf(t, "fast", 9)
	slowDocument := documentsOf(t, "slow", 1)
	peers := peersWhere([]string{"stuck"}, nil)
	graceStarted := make(chan struct{})
	peers.latePeers["slow"] = struct{}{}
	peers.lateAnswersReleased = graceStarted
	asked := askedPeersOf(peers)
	asked.cutoff = cutoffAtNinetyPercent
	asked.clock.graceStarts = func() { close(graceStarted) }

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "fast", documents: fastDocuments},
		peerAbstract{address: "slow", documents: slowDocument},
		peerAbstract{address: "stuck", documents: append(fastDocuments[:1:1], slowDocument...)},
	))

	if performed.EndReason != urlmetadataasks.EndedByCoverage ||
		performed.AmountOfDocumentsCutOff != 0 || asked.clock.gracesStarted != 1 {
		t.Fatalf(
			"the asks reported %+v after %d graces, want them to end by coverage inside "+
				"the one grace",
			performed, asked.clock.gracesStarted,
		)
	}
}

func TestAPeerTheCeilingLimitsIsAskedForTheLeastHeldDocuments(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, "held", 2)
	mostHeld, leastHeld := documents[0], documents[1]
	peers := peersWhere(nil, nil)
	asked := askedPeersOf(peers)
	asked.ceilings = ceilingsOfThePeers{ceilingOfEachPeer: map[string]int{"first": 1}}

	asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: []yacymodel.URLHash{mostHeld, leastHeld}},
		peerAbstract{address: "second", documents: []yacymodel.URLHash{mostHeld}},
	))

	got := map[string][]yacymodel.URLHash{}
	for _, ask := range peers.asks {
		got[ask.Peer.Address] = ask.Documents
	}
	if !slices.Equal(got["first"], []yacymodel.URLHash{leastHeld}) ||
		!slices.Equal(got["second"], []yacymodel.URLHash{mostHeld}) {
		t.Fatalf("the peers were asked for %v, want the least held document of the first", got)
	}
}

func TestEachDocumentIsAskedFromAsManyPeersAsTheAsksPerDocument(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, "joined", 3)
	peers := peersWhere(nil, nil)

	askedPeersOf(peers).settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: documents[:2]},
		peerAbstract{address: "second", documents: documents[1:]},
		peerAbstract{address: "third", documents: []yacymodel.URLHash{documents[0], documents[2]}},
		peerAbstract{address: "fourth", documents: documents},
	))

	asksNamingEachDocument := map[yacymodel.URLHash]int{}
	for _, ask := range peers.asks {
		for _, document := range ask.Documents {
			asksNamingEachDocument[document]++
		}
	}
	for _, document := range documents {
		if asksNamingEachDocument[document] != asksPerDocument {
			t.Fatalf(
				"the peers were asked %v, want each document in %d asks",
				peers.asks,
				asksPerDocument,
			)
		}
	}
}

func TestADocumentIsAskedFromTheHolderWithTheHigherCeiling(t *testing.T) {
	t.Parallel()

	document := documentsOf(t, "joined", 1)
	peers := peersWhere(nil, nil)
	asked := askedPeersOf(peers)
	asked.asksPerDocument = 1
	asked.ceilings = ceilingsOfThePeers{ceilingOfEachPeer: map[string]int{"first": askCeiling / 2}}

	asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: document},
		peerAbstract{address: "second", documents: document},
	))

	if len(peers.asks) != 1 || peers.asks[0].Peer.Address != "second" {
		t.Fatalf(
			"the peers were asked %v, want one ask of the peer with the higher ceiling",
			peers.asks,
		)
	}
}

func TestAJoinGivesADocumentToAHolderItAlreadyAsks(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, "joined", 6)
	peers := peersWhere(nil, nil)
	asked := askedPeersOf(peers)
	asked.asksPerDocument = 1

	asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: documents[:2]},
		peerAbstract{address: "second", documents: documents[1:4]},
		peerAbstract{address: "third", documents: documents[2:]},
	))

	askedAddresses := make([]string, 0, len(peers.asks))
	for _, ask := range peers.asks {
		askedAddresses = append(askedAddresses, ask.Peer.Address)
	}
	slices.Sort(askedAddresses)
	if want := []string{"first", "third"}; !slices.Equal(askedAddresses, want) {
		t.Fatalf("the peers were asked %v, want only the asks of %v", peers.asks, want)
	}
}

func TestEveryPeerHoldingADocumentNoOtherPeerHoldsIsAsked(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, "joined", 4)
	peers := peersWhere(nil, nil)
	asked := askedPeersOf(peers)

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: documents[:1]},
		peerAbstract{address: "second", documents: documents[1:2]},
		peerAbstract{address: "third", documents: documents[2:3]},
		peerAbstract{address: "fourth", documents: documents[3:]},
	))

	if len(peers.asks) != 4 || performed.AmountOfDocumentsNotAsked != 0 {
		t.Fatalf("the peers were asked %v, want four asks with no document left out", peers.asks)
	}
}

func TestTheReportCountsTheDocumentsOfEachAsk(t *testing.T) {
	t.Parallel()

	asked := askedPeersOf(peersWhere(nil, nil))

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: documentsOf(t, "first", 2)},
		peerAbstract{address: "second", documents: documentsOf(t, "second", 3)},
	))

	if !slices.Equal(performed.AmountOfDocumentsPerAsk, []int{2, 3}) ||
		performed.AmountOfLookedUpDocuments != 5 {
		t.Fatalf("the asks reported %+v, want asks of 2 and 3 documents", performed)
	}
}

func TestTheAsksArePutAsSoonAsTheLookupIsAskedWithoutWaitingForTheEnd(t *testing.T) {
	t.Parallel()

	peers := peersWhere(nil, nil)
	asked := askedPeersOf(peers)
	lookup := asked.lookupBegunIn(t.Context())

	lookup.AskFor(holdersIn(answersOf(
		peerAbstract{address: "first", documents: documentsOf(t, "joined", 1)},
	)))
	amountOfAsksPutBeforeTheEnd := len(peers.asks)
	lookup.End()
	performed := asked.reports.performed[0]

	if amountOfAsksPutBeforeTheEnd != 1 || !performed.TimeToFirstAsk.Present() {
		t.Fatalf(
			"%d asks were put before the end and the lookup reported %+v, want one ask put "+
				"and the time to it",
			amountOfAsksPutBeforeTheEnd, performed,
		)
	}
}

func TestALookupThatAskedNoPeerEndsWithEveryAskSettled(t *testing.T) {
	t.Parallel()

	peers := peersWhere(nil, nil)

	performed := askedPeersOf(peers).settledFor(t.Context(), nil)

	if len(peers.asks) != 0 || performed.EndReason != urlmetadataasks.EndedByEveryAskSettled ||
		performed.TimeToFirstAsk.Present() {
		t.Fatalf(
			"the lookup put %v and reported %+v, want no ask, every ask settled and "+
				"no time to a first ask",
			peers.asks, performed,
		)
	}
}

func TestTheAsksOfSeveralJoinsEndOnceTheirAnswersCoverEveryDocument(t *testing.T) {
	t.Parallel()

	laterDocuments := documentsOf(t, "later", 1)
	asked := askedPeersOf(peersWhere([]string{"stuck"}, nil))

	performed := asked.settledFor(
		t.Context(),
		answersOf(peerAbstract{address: "first", documents: documentsOf(t, "earlier", 1)}),
		answersOf(
			peerAbstract{address: "second", documents: laterDocuments},
			peerAbstract{address: "stuck", documents: laterDocuments},
		),
	)

	if performed.EndReason != urlmetadataasks.EndedByCoverage ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 2 {
		t.Fatalf(
			"the asks reported %+v, want them to end by coverage with the documents of both joins",
			performed,
		)
	}
}

func TestTheAsksOfSeveralJoinsEndOnceEveryAskOfEachJoinSettled(t *testing.T) {
	t.Parallel()

	asked := askedPeersOf(peersWhere(nil, []string{"failing"}))

	performed := asked.settledFor(
		t.Context(),
		answersOf(peerAbstract{address: "failing", documents: documentsOf(t, "earlier", 1)}),
		answersOf(peerAbstract{address: "second", documents: documentsOf(t, "later", 1)}),
	)

	if performed.EndReason != urlmetadataasks.EndedByEveryAskSettled ||
		performed.AmountOfLookedUpDocumentsWithMetadata != 1 {
		t.Fatalf(
			"the asks reported %+v, want them to end once the asks of both joins settled",
			performed,
		)
	}
}

func TestTheAsksOfSeveralJoinsAreCutOffAGraceAfterMostOfTheirDocumentsSettled(t *testing.T) {
	t.Parallel()

	asked := askedPeersOf(peersWhere([]string{"stuck"}, nil))
	asked.cutoff = cutoffAtNinetyPercent
	asked.clock.firesAtOnce = true

	performed := asked.settledFor(
		t.Context(),
		answersOf(peerAbstract{address: "fast", documents: documentsOf(t, "fast", 9)}),
		answersOf(peerAbstract{address: "stuck", documents: documentsOf(t, "stuck", 1)}),
	)

	if performed.EndReason != urlmetadataasks.EndedByCutoff ||
		performed.AmountOfDocumentsCutOff != 1 || asked.clock.gracesStarted != 1 {
		t.Fatalf(
			"the asks reported %+v after %d graces, want them cut off after one grace with "+
				"the document of the stuck peer cut off",
			performed, asked.clock.gracesStarted,
		)
	}
}

func TestEachJoinAsksOneHolderOfEachOfItsDocuments(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, "joined", 2)
	peers := peersWhere(nil, nil)
	asked := askedPeersOf(peers)
	asked.asksPerDocument = 1

	performed := asked.settledFor(
		t.Context(),
		answersOf(
			peerAbstract{address: "first", documents: documents[:1]},
			peerAbstract{address: "second", documents: documents[:1]},
		),
		answersOf(
			peerAbstract{address: "third", documents: documents[1:]},
			peerAbstract{address: "fourth", documents: documents[1:]},
		),
	)

	if len(peers.asks) != 2 || performed.AmountOfLookedUpDocuments != 2 {
		t.Fatalf(
			"the peers were asked %v for %d documents, want one ask for each join naming both",
			peers.asks, performed.AmountOfLookedUpDocuments,
		)
	}
}

func TestTheReportCountsTheDocumentsTheLookupWasGivenButNeverAskedAbout(t *testing.T) {
	t.Parallel()

	asked := askedPeersOf(peersWhere(nil, nil))
	asked.ceilings = ceilingsOfThePeers{ceilingOfEachPeer: map[string]int{"first": 1}}

	performed := asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: documentsOf(t, "joined", 3)},
	))

	if performed.AmountOfDocumentsNotAsked != 2 || performed.AmountOfLookedUpDocuments != 1 {
		t.Fatalf(
			"the lookup reported %+v, want one document asked about and two never asked about",
			performed,
		)
	}
}

func TestTheLookupHandsOnWhatEachPeerSentAsItArrivesAndReportsOnceItEnds(t *testing.T) {
	t.Parallel()

	joined := documentsOf(t, "joined", 2)
	asked := askedPeersOf(peersWhere(nil, nil))
	lookup := asked.lookupBegunIn(t.Context())

	lookup.AskFor(holdersIn(answersOf(peerAbstract{address: "first", documents: joined})))
	<-asked.sentMetadata.arrivals
	sentByTheFirstBeforeTheEnd := len(
		asked.sentMetadata.metadataPerPeer[yacymodel.WordHash("first")],
	)
	amountOfReportsBeforeTheEnd := len(asked.reports.performed)
	lookup.End()

	if sentByTheFirstBeforeTheEnd != len(joined) || amountOfReportsBeforeTheEnd != 0 ||
		len(asked.reports.performed) != 1 {
		t.Fatalf(
			"the lookup handed on %d documents and reported %d times before the end and "+
				"%v in all, want the metadata of %d documents from the first peer before "+
				"the end and one report at the end",
			sentByTheFirstBeforeTheEnd, amountOfReportsBeforeTheEnd,
			asked.reports.performed, len(joined),
		)
	}
}

func TestTheGraceStartsAtTheEndWhenMostDocumentsSettledBeforeIt(t *testing.T) {
	t.Parallel()

	asked := askedPeersOf(peersWhere([]string{"stuck"}, nil))
	asked.cutoff = cutoffAtNinetyPercent
	asked.clock.firesAtOnce = true
	lookup := asked.lookupBegunIn(t.Context())

	lookup.AskFor(holdersIn(answersOf(
		peerAbstract{address: "fast", documents: documentsOf(t, "fast", 9)},
	)))
	lookup.AskFor(holdersIn(answersOf(
		peerAbstract{address: "stuck", documents: documentsOf(t, "stuck", 1)},
	)))
	<-asked.sentMetadata.arrivals
	gracesStartedBeforeTheEnd := asked.clock.gracesStarted
	lookup.End()
	performed := asked.reports.performed[0]

	if gracesStartedBeforeTheEnd != 0 || asked.clock.gracesStarted != 1 ||
		performed.EndReason != urlmetadataasks.EndedByCutoff ||
		performed.AmountOfDocumentsCutOff != 1 {
		t.Fatalf(
			"%d graces started before the end and %d in all, and the lookup reported %+v, "+
				"want one grace started at the end and the document of the stuck peer cut off",
			gracesStartedBeforeTheEnd, asked.clock.gracesStarted, performed,
		)
	}
}
