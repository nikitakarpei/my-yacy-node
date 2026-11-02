// Package accountablepeer is the smallest peer that keeps an accountable index: it
// promises what it accepts, turns every removal into a tombstone, seals the epoch's
// changes into one signed head, and answers searches and lookups from its last sealed tree
// with proofs against that head.
package accountablepeer

import (
	"crypto/ed25519"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indexevidence"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indexhead"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/intakepromise"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/provensearch"
)

type Clock func() time.Time

type Identity struct {
	Network string
	Name    string
	Key     ed25519.PrivateKey
}

type Peer struct {
	clock             Clock
	tombstonesByEpoch map[uint64][]indextree.Key
	latest            indexhead.SignedHead
	identity          Identity
	pending           indextree.Changes
	tree              indextree.Tree
	period            time.Duration
	lastReset         uint64
	amountOfWrites    int
}

func New(identity Identity, period time.Duration, clock Clock) *Peer {
	return &Peer{
		clock:             clock,
		tombstonesByEpoch: map[uint64][]indextree.Key{},
		identity:          identity,
		tree:              indextree.EmptyTree(provensearch.SummaryOf),
		period:            period,
	}
}

func (p *Peer) PublicKey() ed25519.PublicKey {
	public, _ := p.identity.Key.Public().(ed25519.PublicKey)

	return public
}

func (p *Peer) Accept(leaves []indextree.Leaf) intakepromise.Promise {
	keys := make([]indextree.Key, 0, len(leaves))
	for _, leaf := range leaves {
		keys = append(keys, leaf.Key)
	}
	p.pending.Stored = append(p.pending.Stored, leaves...)

	return intakepromise.PromiseOf(p.identity.Name, p.currentEpoch()+1, keys, p.identity.Key)
}

func (p *Peer) Remove(keys []indextree.Key, reason provensearch.RemovalReason) {
	epoch := p.currentEpoch()
	for _, key := range keys {
		p.pending.Stored = append(p.pending.Stored, indextree.Leaf{Key: key, Value: provensearch.TombstoneOf(reason, epoch)})
	}
	p.tombstonesByEpoch[epoch] = append(p.tombstonesByEpoch[epoch], keys...)
}

func (p *Peer) Reset() {
	p.tree = indextree.EmptyTree(provensearch.SummaryOf)
	p.pending = indextree.Changes{}
	clear(p.tombstonesByEpoch)
	p.lastReset = p.currentEpoch()
}

func (p *Peer) Seal() indexhead.SignedHead {
	epoch := p.currentEpoch()
	if p.latest.Signature != nil && p.latest.Head.Epoch >= epoch {
		return p.latest
	}
	p.pending.Removed = append(p.pending.Removed, p.expiredTombstonesAt(epoch)...)
	var written int
	p.tree, written = p.tree.Updated(p.pending)
	p.amountOfWrites += written
	p.pending = indextree.Changes{}
	p.latest = indexhead.SignedHeadOf(indexhead.Head{
		Network:   p.identity.Network,
		Peer:      p.identity.Name,
		Epoch:     epoch,
		LastReset: p.lastReset,
		Previous:  p.previousHeadHash(),
		Root:      p.tree.Root(),
		Summary:   p.tree.Summary(),
	}, p.identity.Key)

	return p.latest
}

func (p *Peer) expiredTombstonesAt(epoch uint64) []indextree.Key {
	var expired []indextree.Key
	for removalEpoch, keys := range p.tombstonesByEpoch {
		if removalEpoch+indexevidence.TombstoneEpochs >= epoch {
			continue
		}
		for _, key := range keys {
			value, _ := p.tree.Value(key)
			if tombstoneEpoch, isTombstone := provensearch.TombstoneEpochOf(value); isTombstone && tombstoneEpoch == removalEpoch {
				expired = append(expired, key)
			}
		}
		delete(p.tombstonesByEpoch, removalEpoch)
	}

	return expired
}

func (p *Peer) previousHeadHash() indextree.Hash {
	if p.latest.Signature == nil {
		return indextree.Hash{}
	}

	return p.latest.Head.Hash()
}

func (p *Peer) currentEpoch() uint64 {
	return indexhead.EpochAt(p.clock(), p.period)
}

func (p *Peer) AmountOfWrittenNodes() int {
	return p.amountOfWrites
}

func (p *Peer) Search(query provensearch.Query, amountOfExpansions int) (indexhead.SignedHead, provensearch.Answer) {
	return p.latest, provensearch.AnswerFrom(p.tree, query, amountOfExpansions)
}

func (p *Peer) Lookup(keys []indextree.Key) indexevidence.Observation {
	prover := p.tree.Prover()
	for _, key := range keys {
		prover.RevealLookupOf(key)
	}

	return indexevidence.Observation{Head: p.latest, Proof: prover.Proof()}
}

func (p *Peer) CountProof(word indextree.Word, rank uint64) indexevidence.Observation {
	prover := p.tree.Prover()
	prover.RevealRankOf(word, rank)

	return indexevidence.Observation{Head: p.latest, Proof: prover.Proof()}
}
