// Package indexevidence turns signed heads, promises and proofs into evidence that a peer
// broke the accountable index contract. Evidence needs nothing but the peer's public key to
// check, and every rule only fires on facts the peer itself committed to. A reset head
// discloses the removal of every key the peer held before it.
package indexevidence

import (
	"crypto/ed25519"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indexhead"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/intakepromise"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/provensearch"
)

const TombstoneEpochs = 144

type Observation struct {
	Proof indextree.Proof
	Head  indexhead.SignedHead
}

func (o Observation) isProvenBy(key ed25519.PublicKey) bool {
	return o.Head.IsSignedBy(key) &&
		provensearch.VerifiedProof(o.Proof, o.Head.Head.Root, o.Head.Head.Summary) == nil
}

func (o Observation) epoch() uint64 {
	return o.Head.Head.Epoch
}

type coverage int

const (
	uncovered coverage = iota
	covered
	unproven
)

func (o Observation) coverageOf(key indextree.Key, removedAfter uint64) coverage {
	held, presence := o.Proof.Lookup(key)
	switch {
	case presence == indextree.Unproven:
		return unproven
	case presence == indextree.Present && held.Summary.Live == 1, o.Head.Head.LastReset > removedAfter:
		return covered
	}
	result := uncovered
	for _, tombstoneKey := range TombstoneKeysOf(key) {
		switch o.tombstoneCoverageOf(tombstoneKey, removedAfter) {
		case covered:
			return covered
		case unproven:
			result = unproven
		case uncovered:
		}
	}

	return result
}

func (o Observation) tombstoneCoverageOf(tombstoneKey indextree.Key, removedAfter uint64) coverage {
	tombstone, presence := o.Proof.Lookup(tombstoneKey)
	switch {
	case presence == indextree.Absent:
		return uncovered
	case presence == indextree.Unproven || tombstone.Leaf == nil:
		return unproven
	}
	if removal, isURLTombstone := provensearch.URLTombstoneFrom(tombstone.Leaf.Value); isURLTombstone && removal.Epoch > removedAfter {
		return covered
	}
	if handoff, isWordTombstone := provensearch.WordTombstoneFrom(tombstone.Leaf.Value); isWordTombstone && handoff.Epoch > removedAfter {
		return covered
	}

	return uncovered
}

func TombstoneKeysOf(key indextree.Key) []indextree.Key {
	return []indextree.Key{
		provensearch.RemovedURLKeyOf(key.URL()),
		provensearch.HandedOffPartitionKeyOf(key.Word(), provensearch.PartitionOf(key.URL())),
	}
}

type BrokenPromise struct {
	Promise     intakepromise.Promise
	Observation Observation
	Key         indextree.Key
}

func BrokenPromiseOf(promise intakepromise.Promise, observation Observation, key ed25519.PublicKey) (BrokenPromise, bool) {
	if !promise.IsSignedBy(key) || !observation.isProvenBy(key) || observation.Head.Head.Peer != promise.Peer ||
		observation.epoch() < promise.Epoch || observation.epoch() > promise.Epoch+TombstoneEpochs {
		return BrokenPromise{}, false
	}
	for _, promised := range promise.Keys {
		if observation.coverageOf(promised, promise.Epoch-1) == uncovered {
			return BrokenPromise{Promise: promise, Observation: observation, Key: promised}, true
		}
	}

	return BrokenPromise{}, false
}

type UndisclosedRemoval struct {
	Earlier Observation
	Later   Observation
	Key     indextree.Key
}

func UndisclosedRemovalOf(earlier, later Observation, removed indextree.Key, key ed25519.PublicKey) (UndisclosedRemoval, bool) {
	if !areOrderedObservations(earlier, later, key) {
		return UndisclosedRemoval{}, false
	}
	held, presence := earlier.Proof.Lookup(removed)
	if presence != indextree.Present || held.Summary.Live != 1 {
		return UndisclosedRemoval{}, false
	}
	if later.coverageOf(removed, earlier.epoch()) != uncovered {
		return UndisclosedRemoval{}, false
	}

	return UndisclosedRemoval{Earlier: earlier, Later: later, Key: removed}, true
}

func areOrderedObservations(earlier, later Observation, key ed25519.PublicKey) bool {
	return earlier.isProvenBy(key) && later.isProvenBy(key) &&
		earlier.Head.Head.Peer == later.Head.Head.Peer &&
		later.epoch() > earlier.epoch() && later.epoch() <= earlier.epoch()+TombstoneEpochs
}
