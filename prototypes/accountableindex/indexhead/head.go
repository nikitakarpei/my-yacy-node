// Package indexhead holds the signed head of a peer's posting tree for one epoch and the
// rule that turns two heads into fork evidence. The epoch is the index of a fixed period of
// UTC time, and each head names the epoch of its predecessor, so two chains of one peer
// cannot both pass an observer that holds a head of each.
package indexhead

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

const (
	headDomain    = "yacy-index-head"
	FormatVersion = 1
)

type Head struct {
	Network          string
	Peer             string
	CertificateSeq   uint64
	Epoch            uint64
	PredecessorEpoch uint64
	Summary          indextree.Summary
	Predecessor      indextree.Hash
	Root             indextree.Hash
	EvictionCutoff   uint16
}

func (h Head) Bytes() []byte {
	out := appendText(nil, headDomain)
	out = binary.BigEndian.AppendUint16(out, FormatVersion)
	out = appendText(out, h.Network)
	out = appendText(out, h.Peer)
	out = binary.BigEndian.AppendUint64(out, h.CertificateSeq)
	out = binary.BigEndian.AppendUint64(out, h.Epoch)
	out = binary.BigEndian.AppendUint64(out, h.PredecessorEpoch)
	out = append(out, h.Predecessor[:]...)
	out = append(out, h.Root[:]...)
	out = binary.BigEndian.AppendUint64(out, h.Summary.Live)
	out = binary.BigEndian.AppendUint64(out, h.Summary.Removed)
	out = binary.BigEndian.AppendUint16(out, h.Summary.TopImpact)
	out = binary.BigEndian.AppendUint16(out, h.Summary.TopStaleness)

	return binary.BigEndian.AppendUint16(out, h.EvictionCutoff)
}

func appendText(out []byte, text string) []byte {
	out = binary.BigEndian.AppendUint16(out, uint16(len(text)))

	return append(out, text...)
}

func (h Head) Hash() indextree.Hash {
	return sha256.Sum256(h.Bytes())
}

type SignedHead struct {
	Signature []byte
	Head      Head
}

func SignedHeadOf(head Head, key ed25519.PrivateKey) SignedHead {
	return SignedHead{Head: head, Signature: ed25519.Sign(key, head.Bytes())}
}

func (s SignedHead) IsSignedBy(key ed25519.PublicKey) bool {
	return ed25519.Verify(key, s.Head.Bytes(), s.Signature)
}

func EpochAt(moment time.Time, period time.Duration) uint64 {
	return uint64(moment.UTC().UnixNano() / int64(period))
}

type Fork struct {
	Earlier SignedHead
	Later   SignedHead
}

func ForkOf(earlier, later SignedHead, key ed25519.PublicKey) (Fork, bool) {
	if !earlier.IsSignedBy(key) || !later.IsSignedBy(key) || earlier.Head.Peer != later.Head.Peer ||
		earlier.Head.Epoch > later.Head.Epoch {
		return Fork{}, false
	}
	fork := Fork{Earlier: earlier, Later: later}
	if earlier.Head.Epoch == later.Head.Epoch {
		return fork, earlier.Head.Hash() != later.Head.Hash()
	}
	if later.Head.PredecessorEpoch < earlier.Head.Epoch {
		return fork, true
	}

	return fork, later.Head.PredecessorEpoch == earlier.Head.Epoch && later.Head.Predecessor != earlier.Head.Hash()
}
