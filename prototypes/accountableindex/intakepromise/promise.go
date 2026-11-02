// Package intakepromise holds the signed promise a peer gives when it accepts postings:
// each accepted key appears in its heads from a named epoch on, as a posting or as a
// tombstone that discloses its removal.
package intakepromise

import (
	"crypto/ed25519"
	"encoding/binary"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

const promiseDomain = "yacy-index-promise/1"

type Promise struct {
	Peer      string
	Keys      []indextree.Key
	Signature []byte
	Epoch     uint64
}

func PromiseOf(peer string, epoch uint64, keys []indextree.Key, key ed25519.PrivateKey) Promise {
	sortedKeys := slices.Clone(keys)
	slices.SortFunc(sortedKeys, func(a, b indextree.Key) int { return slices.Compare(a[:], b[:]) })
	promise := Promise{Peer: peer, Epoch: epoch, Keys: sortedKeys}
	promise.Signature = ed25519.Sign(key, promise.signedBytes())

	return promise
}

func (p Promise) signedBytes() []byte {
	out := binary.BigEndian.AppendUint16(nil, uint16(len(promiseDomain)))
	out = append(out, promiseDomain...)
	out = binary.BigEndian.AppendUint16(out, uint16(len(p.Peer)))
	out = append(out, p.Peer...)
	out = binary.BigEndian.AppendUint64(out, p.Epoch)
	for _, key := range p.Keys {
		out = append(out, key[:]...)
	}

	return out
}

func (p Promise) IsSignedBy(key ed25519.PublicKey) bool {
	return ed25519.Verify(key, p.signedBytes(), p.Signature)
}
