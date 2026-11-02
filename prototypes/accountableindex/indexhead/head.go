// Package indexhead holds the signed head of a peer's index tree for one epoch and the
// rules that turn two heads into fork evidence. The epoch is the index of a fixed period
// of UTC time, so two chains of one peer must use the same epoch numbers.
package indexhead

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

const headDomain = "yacy-index-head/1"

type Head struct {
	Network   string
	Peer      string
	Epoch     uint64
	LastReset uint64
	Summary   indextree.Summary
	Previous  indextree.Hash
	Root      indextree.Hash
}

func (h Head) Bytes() []byte {
	out := appendText(nil, headDomain)
	out = appendText(out, h.Network)
	out = appendText(out, h.Peer)
	out = binary.BigEndian.AppendUint64(out, h.Epoch)
	out = binary.BigEndian.AppendUint64(out, h.LastReset)
	out = append(out, h.Previous[:]...)
	out = append(out, h.Root[:]...)
	out = binary.BigEndian.AppendUint64(out, h.Summary.Live)
	out = binary.BigEndian.AppendUint64(out, h.Summary.Removed)
	out = binary.BigEndian.AppendUint16(out, h.Summary.TopImpact)

	return out
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

func (s SignedHead) IsCurrentAt(moment time.Time, period time.Duration) bool {
	current := EpochAt(moment, period)

	return s.Head.Epoch+1 >= current && s.Head.Epoch <= current+1
}

type Fork struct {
	Earlier SignedHead
	Later   SignedHead
}

func ForkOf(earlier, later SignedHead, key ed25519.PublicKey) (Fork, bool) {
	if !earlier.IsSignedBy(key) || !later.IsSignedBy(key) || earlier.Head.Peer != later.Head.Peer {
		return Fork{}, false
	}
	sameEpoch := earlier.Head.Epoch == later.Head.Epoch && earlier.Head.Hash() != later.Head.Hash()
	brokenChain := later.Head.Epoch == earlier.Head.Epoch+1 && later.Head.Previous != earlier.Head.Hash()

	return Fork{Earlier: earlier, Later: later}, sameEpoch || brokenChain
}
