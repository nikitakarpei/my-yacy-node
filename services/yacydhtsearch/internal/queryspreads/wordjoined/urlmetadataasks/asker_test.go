package urlmetadataasks_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined/urlmetadataasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	askCeiling          = 20
	peersHoldingOneWord = 24
	grace               = time.Minute
)

var cutoffAtNinetyPercent = urlmetadataasks.Cutoff{PercentOfDocuments: 90, Grace: grace}

type peersOfTheNetwork struct {
	stuckPeers   map[string]struct{}
	failingPeers map[string]struct{}
	asks         []peerasks.URLMetadataAsk
	asksContext  context.Context
}

func peersWhere(stuckPeers []string, failingPeers []string) *peersOfTheNetwork {
	peers := &peersOfTheNetwork{
		stuckPeers:   map[string]struct{}{},
		failingPeers: map[string]struct{}{},
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
	for _, ask := range asks {
		if _, stuck := peers.stuckPeers[ask.Peer.Address]; stuck {
			anyStuck = true

			continue
		}
		outcomesAsTheySettle <- peers.outcomeOf(ask)
	}
	if !anyStuck {
		close(outcomesAsTheySettle)

		return outcomesAsTheySettle
	}
	go func() {
		<-ctx.Done()
		close(outcomesAsTheySettle)
	}()

	return outcomesAsTheySettle
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
}

func (clock *graceClock) After(_ time.Duration, expire func()) func() {
	clock.gracesStarted++
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

type askedPeers struct {
	peers      *peersOfTheNetwork
	ceilings   ceilingsOfThePeers
	cutoff     urlmetadataasks.Cutoff
	clock      *graceClock
	holdingOne int
}

func askedPeersOf(peers *peersOfTheNetwork) askedPeers {
	return askedPeers{peers: peers, clock: &graceClock{}, holdingOne: peersHoldingOneWord}
}

func (asked askedPeers) settledFor(
	ctx context.Context,
	answers []wordpartitionasks.ReplicaAnswer,
) urlmetadataasks.Performed {
	return urlmetadataasks.PerformedFrom(urlmetadataasks.New(
		asked.peers, asked.ceilings, asked.cutoff, asked.clock, asked.holdingOne,
	).AskFor(ctx, documentsAcross(answers), answers))
}

func documentsAcross(answers []wordpartitionasks.ReplicaAnswer) []yacymodel.URLHash {
	var documents []yacymodel.URLHash
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if slices.Contains(documents, listedDocument.Hash) {
				continue
			}
			documents = append(documents, listedDocument.Hash)
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
		performed.AmountOfAskedDocumentsWithMetadata != 1 {
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
		performed.AmountOfAskedDocumentsWithMetadata != 2 {
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
		performed.AmountOfAskedDocumentsWithMetadata != 9 {
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
		performed.AmountOfAskedDocumentsWithMetadata != 16 {
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
		performed.AmountOfAskedDocumentsWithMetadata != 9 {
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
	asked := askedPeersOf(peersWhere([]string{"stuck"}, nil))
	asked.cutoff = cutoffAtNinetyPercent

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

func TestNoMorePeersAreAskedThanHoldOneWordAndTheyCoverEveryDocument(t *testing.T) {
	t.Parallel()

	documents := documentsOf(t, "joined", 2)
	peers := peersWhere(nil, nil)
	asked := askedPeersOf(peers)
	asked.holdingOne = 2

	asked.settledFor(t.Context(), answersOf(
		peerAbstract{address: "first", documents: documents[:1]},
		peerAbstract{address: "second", documents: documents[:1]},
		peerAbstract{address: "third", documents: documents[1:]},
		peerAbstract{address: "fourth", documents: documents[1:]},
	))

	var askedDocuments []yacymodel.URLHash
	for _, ask := range peers.asks {
		askedDocuments = append(askedDocuments, ask.Documents...)
	}
	slices.SortFunc(askedDocuments, func(first, second yacymodel.URLHash) int {
		return strings.Compare(first.String(), second.String())
	})
	slices.SortFunc(documents, func(first, second yacymodel.URLHash) int {
		return strings.Compare(first.String(), second.String())
	})
	if len(peers.asks) != 2 || !slices.Equal(askedDocuments, documents) {
		t.Fatalf("the peers were asked %v, want two asks covering %v", peers.asks, documents)
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
		performed.AmountOfAskedDocuments != 5 {
		t.Fatalf("the asks reported %+v, want asks of 2 and 3 documents", performed)
	}
}
