package provenpostingoffer_test

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestAnOfferAfterAPartialHandoffProvesThePostingsToTheirRoot(t *testing.T) {
	node := openNode(t)
	crawled := node.crawledPagePostingsFrom(t, "w1", "w2", "w3", "w4")
	node.write(t, crawled)
	node.handOff(t, crawled.Postings()[1])

	offeredPostings := []yacymodel.RWIPosting{crawled.Postings()[0], crawled.Postings()[3]}
	received := node.offer(t, offeredPostings)

	proof, proven := received.PostingsProofs[pageURLHash]
	if !proven {
		t.Fatalf("offer carried no proof for %v", pageURLHash)
	}
	receivedPagePostings, err := yacymodel.ProvenPagePostingsFrom(received.Indexes, proof)
	if err != nil {
		t.Fatal(err)
	}
	if !crawled.Root().Present() || receivedPagePostings.Root() != crawled.Root() {
		t.Fatalf("root of the offered postings = %+v, want %+v",
			receivedPagePostings.Root(), crawled.Root())
	}
}

func TestAnOfferSendsThePostingsWithoutPostingsProofsWhenTheHeldProofsCannotBeRead(
	t *testing.T,
) {
	node := openNode(t)
	crawled := node.crawledPagePostingsFrom(t, "w1", "w2")
	node.write(t, crawled)
	if proven := node.offer(t, crawled.Postings()).PostingsProofs; len(proven) != 1 {
		t.Fatalf("offer before the read failure carried proofs %v, want one", proven)
	}

	node.engine.failViews()
	received := node.offer(t, crawled.Postings())

	if len(received.Indexes) != len(crawled.Postings()) || len(received.PostingsProofs) != 0 {
		t.Fatalf("offer carried %d postings and proofs %v, want %d postings and no proofs",
			len(received.Indexes), received.PostingsProofs, len(crawled.Postings()))
	}
}

type node struct {
	vault      *vault.Vault
	engine     *engineWithFailingViews
	partitions yacymodel.DHTRingPartitions
	writer     rwipostings.PostingWriter
	purger     rwipostings.PostingPurger
	courier    postingcourier.Courier
	peer       *identifiedPeer
}

func openNode(t *testing.T) node {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}
	engine := &engineWithFailingViews{Engine: memoryvault.OpenEngine(0)}
	v, err := vault.New(engine, nil, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	_, writer, purger, proofs, err := rwipostings.Open(t.Context(), v, partitions, nil, nil, nil)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	peer := &identifiedPeer{identity: peerkeytest.Key(0).Identity()}

	return node{
		vault:      v,
		engine:     engine,
		partitions: partitions,
		writer:     writer,
		purger:     purger,
		courier: postingcourier.New(
			peer,
			peer,
			proofs,
			"freeworld",
			peerkeytest.Key(1).Identity().Hash(),
			peer,
		),
		peer: peer,
	}
}

var errViewFailed = errors.New("view failed")

type engineWithFailingViews struct {
	vault.Engine

	viewsFail bool
}

func (e *engineWithFailingViews) View(ctx context.Context, fn func(vault.EngineTxn) error) error {
	if e.viewsFail {
		return errViewFailed
	}

	return e.Engine.View(ctx, fn)
}

func (e *engineWithFailingViews) failViews() {
	e.viewsFail = true
}

type identifiedPeer struct {
	identity      yacymodel.PeerIdentity
	receivedForms []url.Values
}

func (p *identifiedPeer) RecordOf(
	_ context.Context,
	_ yacymodel.Hash,
) (peerroster.PeerRecord, bool) {
	return peerroster.PeerRecord{
		Callee:          peerwire.IdentifiedCallee(p.identity),
		AnsweredBaseURL: yacymodel.Some(peerwire.BaseURL{Scheme: peerwire.HTTP}),
	}, true
}

func (p *identifiedPeer) TryAt(
	_ context.Context,
	request peerwire.Request,
	_ peerwire.BaseURL,
	_ func(yacyproto.ResponseFields) bool,
	_ time.Duration,
) (peerwire.Answer, bool) {
	p.receivedForms = append(p.receivedForms, request.Form)

	return peerwire.Answer{
		Fields: yacyproto.TransferRWIResponse{Result: yacyproto.ResultOK}.Encode(),
	}, true
}

func (*identifiedPeer) ObservePostingOfferAnswerTime(time.Duration) {}

func (n node) crawledPagePostingsFrom(t *testing.T, words ...string) yacymodel.PagePostings {
	t.Helper()
	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		postings = append(postings, yacymodel.RWIPosting{
			WordHash: yacymodel.WordHash(word),
			URLHash:  pageURLHash,
			Language: yacymodel.LanguageOfUndeclaredDocument,
			Hits:     1,
		})
	}

	crawled, err := yacymodel.ProvenPagePostingsFrom(
		postings,
		yacymodel.EmptyPostingsProof(n.partitions),
	)
	if err != nil {
		t.Fatal(err)
	}

	return crawled
}

var pageURLHash = func() yacymodel.URLHash {
	address, err := url.Parse("http://example.com/page")
	if err != nil {
		panic(err)
	}

	return yacymodel.URLHashOf(address)
}()

func (n node) write(t *testing.T, pagePostings yacymodel.PagePostings) {
	t.Helper()

	n.update(t, func(tx *vault.Txn) error { return n.writer.Write(tx, pagePostings) })
}

func (n node) update(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := n.vault.Update(t.Context(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (n node) handOff(t *testing.T, posting yacymodel.RWIPosting) {
	t.Helper()

	n.update(t, func(tx *vault.Txn) error {
		_, err := n.purger.PurgePostingsOfWordIn(
			tx,
			posting.WordHash,
			n.partitions.PartitionOf(posting.URLHash),
		)

		return err
	})
}

func (n node) offer(t *testing.T, postings []yacymodel.RWIPosting) yacyproto.TransferRWIRequest {
	t.Helper()

	formsBefore := len(n.peer.receivedForms)
	n.courier.Offer(t.Context(), n.peer.identity.Hash(), postings)
	if len(n.peer.receivedForms) != formsBefore+1 {
		t.Fatalf("peer received %d offers, want 1", len(n.peer.receivedForms)-formsBefore)
	}
	received, err := yacyproto.ParseTransferRWIRequest(
		t.Context(),
		n.peer.receivedForms[formsBefore],
	)
	if err != nil {
		t.Fatalf("ParseTransferRWIRequest: %v", err)
	}

	return received
}
