// Package accountablepeer is the smallest peer that keeps an accountable index: it
// promises what it admits, discloses every removal with a tombstone per URL or per handed-off
// word, evicts by a published cutoff day, seals the epoch's changes into one signed head,
// and answers searches and lookups from its last sealed tree with proofs against that head.
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
	clock           Clock
	keysByURL       map[indextree.URL][]indextree.Key
	tombstoneEpochs map[indextree.Key]uint64
	latest          indexhead.SignedHead
	identity        Identity
	pending         indextree.Changes
	tree            indextree.Tree
	period          time.Duration
	evictionCutoff  uint16
}

func New(identity Identity, period time.Duration, clock Clock) *Peer {
	return &Peer{
		clock:           clock,
		keysByURL:       map[indextree.URL][]indextree.Key{},
		tombstoneEpochs: map[indextree.Key]uint64{},
		identity:        identity,
		tree:            indextree.EmptyTree(provensearch.SummaryOf),
		period:          period,
	}
}

func (p *Peer) PublicKey() ed25519.PublicKey {
	public, _ := p.identity.Key.Public().(ed25519.PublicKey)

	return public
}

func (p *Peer) Admit(leaves []indextree.Leaf) intakepromise.Promise {
	keys := make([]indextree.Key, 0, len(leaves))
	for _, leaf := range leaves {
		keys = append(keys, leaf.Key)
		p.keysByURL[leaf.Key.URL()] = append(p.keysByURL[leaf.Key.URL()], leaf.Key)
	}
	p.pending.Stored = append(p.pending.Stored, leaves...)

	return intakepromise.PromiseOf(p.identity.Name, p.disclosingEpoch(), keys, p.identity.Key)
}

func (p *Peer) RemoveURL(url indextree.URL, reason provensearch.RemovalReason, versionDay uint16) {
	p.pending.Removed = append(p.pending.Removed, p.keysByURL[url]...)
	delete(p.keysByURL, url)
	p.storeTombstone(provensearch.RemovedURLKeyOf(url), provensearch.URLTombstoneOf(provensearch.URLTombstone{
		Reason: reason, Epoch: p.disclosingEpoch(), VersionDay: versionDay, EvictionCutoff: p.evictionCutoff,
	}))
}

func (p *Peer) Evict(cutoffDay uint16) {
	p.evictionCutoff = cutoffDay
	for url, keys := range p.keysByURL {
		value, _ := p.valueOf(keys[0])
		if posting, isPosting := provensearch.PostingFrom(value); isPosting && posting.Day <= cutoffDay {
			p.RemoveURL(url, provensearch.RemovedByEviction, posting.Day)
		}
	}
}

func (p *Peer) HandOff(word indextree.Word, receivers []string) {
	for url, keys := range p.keysByURL {
		kept := keys[:0]
		for _, key := range keys {
			if key.Word() == word {
				p.pending.Removed = append(p.pending.Removed, key)
			} else {
				kept = append(kept, key)
			}
		}
		p.keysByURL[url] = kept
	}
	p.storeTombstone(provensearch.HandedOffWordKeyOf(word), provensearch.WordTombstoneOf(provensearch.WordTombstone{
		Epoch: p.disclosingEpoch(), Receivers: receivers,
	}))
}

func (p *Peer) storeTombstone(key indextree.Key, value []byte) {
	p.pending.Stored = append(p.pending.Stored, indextree.Leaf{Key: key, Value: value})
	p.tombstoneEpochs[key] = p.disclosingEpoch()
}

func (p *Peer) disclosingEpoch() uint64 {
	return p.currentEpoch() + 1
}

func (p *Peer) valueOf(key indextree.Key) ([]byte, bool) {
	for index := len(p.pending.Stored) - 1; index >= 0; index-- {
		if p.pending.Stored[index].Key == key {
			return p.pending.Stored[index].Value, true
		}
	}

	return p.tree.Value(key)
}

func (p *Peer) Seal() indexhead.SignedHead {
	epoch := p.currentEpoch()
	if p.latest.Signature != nil && p.latest.Head.Epoch >= epoch {
		return p.latest
	}
	p.pending.Removed = append(p.pending.Removed, p.expiredTombstonesAt(epoch)...)
	p.tree, _ = p.tree.Updated(p.pending)
	p.pending = indextree.Changes{}
	p.latest = indexhead.SignedHeadOf(indexhead.Head{
		Network:          p.identity.Network,
		Peer:             p.identity.Name,
		Epoch:            epoch,
		PredecessorEpoch: p.latest.Head.Epoch,
		Predecessor:      p.predecessorHash(),
		Root:             p.tree.Root(),
		Summary:          p.tree.Summary(),
		EvictionCutoff:   p.evictionCutoff,
	}, p.identity.Key)

	return p.latest
}

func (p *Peer) expiredTombstonesAt(epoch uint64) []indextree.Key {
	var expired []indextree.Key
	for key, removalEpoch := range p.tombstoneEpochs {
		if removalEpoch+indexevidence.TombstoneEpochs < epoch {
			expired = append(expired, key)
			delete(p.tombstoneEpochs, key)
		}
	}

	return expired
}

func (p *Peer) predecessorHash() indextree.Hash {
	if p.latest.Signature == nil {
		return indextree.Hash{}
	}

	return p.latest.Head.Hash()
}

func (p *Peer) currentEpoch() uint64 {
	return indexhead.EpochAt(p.clock(), p.period)
}

func (p *Peer) Search(query provensearch.Query, amountOfRounds int) (indexhead.SignedHead, provensearch.Answer) {
	return p.latest, provensearch.AnswerFrom(p.tree, query, amountOfRounds)
}

func (p *Peer) Lookup(keys []indextree.Key) indexevidence.Observation {
	prover := p.tree.Prover()
	for _, key := range keys {
		prover.RevealLookupOf(key)
		prover.RevealLookupOf(provensearch.RemovedURLKeyOf(key.URL()))
		prover.RevealLookupOf(provensearch.HandedOffWordKeyOf(key.Word()))
	}

	return indexevidence.Observation{Head: p.latest, Proof: prover.Proof()}
}
