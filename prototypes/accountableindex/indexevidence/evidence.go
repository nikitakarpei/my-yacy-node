// Package indexevidence turns signed heads, promises and proofs into evidence that a peer
// broke the accountable index contract: a promised key that neither holds a posting nor a
// tombstone, or a key that disappears without a tombstone inside the tombstone window.
// Evidence needs nothing but the peer's public key to check.
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

type BrokenPromise struct {
	Promise     intakepromise.Promise
	Observation Observation
	Key         indextree.Key
}

func BrokenPromiseOf(
	promise intakepromise.Promise,
	observation Observation,
	key ed25519.PublicKey,
) (BrokenPromise, bool) {
	head := observation.Head.Head
	if !promise.IsSignedBy(key) || !observation.isProvenBy(key) || head.Peer != promise.Peer ||
		head.Epoch < promise.Epoch || head.Epoch > promise.Epoch+TombstoneEpochs || head.LastReset >= promise.Epoch {
		return BrokenPromise{}, false
	}
	for _, promised := range promise.Keys {
		if _, presence := observation.Proof.Lookup(promised); presence == indextree.Absent {
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

func UndisclosedRemovalOf(
	earlier, later Observation,
	removed indextree.Key,
	key ed25519.PublicKey,
) (UndisclosedRemoval, bool) {
	if !earlier.isProvenBy(key) || !later.isProvenBy(key) ||
		earlier.Head.Head.Peer != later.Head.Head.Peer ||
		later.Head.Head.Epoch <= earlier.Head.Head.Epoch || later.Head.Head.LastReset > earlier.Head.Head.Epoch {
		return UndisclosedRemoval{}, false
	}
	held, heldPresence := earlier.Proof.Lookup(removed)
	_, laterPresence := later.Proof.Lookup(removed)
	if heldPresence != indextree.Present || laterPresence != indextree.Absent {
		return UndisclosedRemoval{}, false
	}
	deadline, isKnown := disclosureDeadlineOf(held, earlier.Head.Head.Epoch)
	if !isKnown || later.Head.Head.Epoch > deadline {
		return UndisclosedRemoval{}, false
	}

	return UndisclosedRemoval{Earlier: earlier, Later: later, Key: removed}, true
}

func disclosureDeadlineOf(held *indextree.ProofNode, heldEpoch uint64) (uint64, bool) {
	if held.Summary.Live == 1 {
		return heldEpoch + TombstoneEpochs, true
	}
	if held.Leaf == nil {
		return 0, false
	}
	removalEpoch, isTombstone := provensearch.TombstoneEpochOf(held.Leaf.Value)

	return removalEpoch + TombstoneEpochs, isTombstone
}
