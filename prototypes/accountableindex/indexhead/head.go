// Package indexhead holds the signed head of a peer's sealed posting tree: the root and
// summary a searcher checks a proven answer against, numbered by a sequence that strictly
// increases with every seal.
package indexhead

import (
	"crypto/ed25519"
	"encoding/binary"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

const (
	headDomain    = "yacy-index-head"
	FormatVersion = 1
)

type Head struct {
	Network  string
	Peer     string
	Sequence uint64
	Summary  indextree.Summary
	Root     indextree.Hash
}

func (h Head) Bytes() []byte {
	out := appendText(nil, headDomain)
	out = binary.BigEndian.AppendUint16(out, FormatVersion)
	out = appendText(out, h.Network)
	out = appendText(out, h.Peer)
	out = binary.BigEndian.AppendUint64(out, h.Sequence)
	out = append(out, h.Root[:]...)
	out = binary.BigEndian.AppendUint64(out, h.Summary.Postings)

	return binary.BigEndian.AppendUint16(out, h.Summary.TopImpact)
}

func appendText(out []byte, text string) []byte {
	out = binary.BigEndian.AppendUint16(out, uint16(len(text)))

	return append(out, text...)
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
