package indextree

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrBadProof = errors.New("bad proof")

type ProofNode struct {
	Leaf     *Leaf
	Left     *ProofNode
	Right    *ProofNode
	Summary  Summary
	Length   int
	Hash     Hash
	Prefix   Key
	Revealed bool
}

type Proof struct {
	Root *ProofNode
}

func (p Proof) VerifiedRoot(summaryOf SummaryRule) (Hash, Summary, error) {
	if p.Root == nil {
		return Hash{}, Summary{}, nil
	}
	if err := p.Root.verify(summaryOf); err != nil {
		return Hash{}, Summary{}, err
	}

	return p.Root.Hash, p.Root.Summary, nil
}

func (n *ProofNode) verify(summaryOf SummaryRule) error {
	if n.Prefix != n.Prefix.prefixOfLength(n.Length) || n.Length < 0 || n.Length > keyBits {
		return fmt.Errorf("%w: range of %d bits is not canonical", ErrBadProof, n.Length)
	}
	switch {
	case !n.Revealed:
		if n.Length == keyBits && n.Summary.Postings != 1 {
			return fmt.Errorf("%w: a leaf stub holds %d postings", ErrBadProof, n.Summary.Postings)
		}

		return nil
	case n.Leaf != nil:
		return n.verifyLeaf(summaryOf)
	default:
		return n.verifyInner(summaryOf)
	}
}

func (n *ProofNode) verifyLeaf(summaryOf SummaryRule) error {
	if n.Length != keyBits || n.Prefix != n.Leaf.Key {
		return fmt.Errorf("%w: leaf sits outside its key", ErrBadProof)
	}
	n.Hash = leafHashOf(n.Leaf.Key, n.Leaf.Value)
	n.Summary = summaryOf(n.Leaf.Value)

	return nil
}

func (n *ProofNode) verifyInner(summaryOf SummaryRule) error {
	if n.Left == nil || n.Right == nil || n.Length >= keyBits {
		return fmt.Errorf("%w: inner node without two children", ErrBadProof)
	}
	for side, child := range []*ProofNode{n.Left, n.Right} {
		if child.Length <= n.Length || !child.Prefix.sharesPrefixWith(n.Prefix, n.Length) ||
			child.Prefix.bitAt(n.Length) != byte(side) {
			return fmt.Errorf("%w: child outside its branch", ErrBadProof)
		}
		if err := child.verify(summaryOf); err != nil {
			return err
		}
	}
	n.Hash = innerHashOf(n.Length, n.Prefix, n.Left.Hash, n.Left.Summary, n.Right.Hash, n.Right.Summary)
	n.Summary = n.Left.Summary.combinedWith(n.Right.Summary)

	return nil
}

type Presence int

const (
	Absent Presence = iota
	Present
	Unproven
)

func (p Proof) RegionOf(word Word) (*ProofNode, Presence) {
	return p.Root.descendedToward(wordPrefixOf(word), wordBits)
}

func (p Proof) Lookup(key Key) (*ProofNode, Presence) {
	return p.Root.descendedToward(key, keyBits)
}

func (n *ProofNode) descendedToward(key Key, length int) (*ProofNode, Presence) {
	current := n
	for current != nil {
		if !key.sharesPrefixWith(current.Prefix, min(current.Length, length)) {
			return nil, Absent
		}
		if current.Length >= length {
			return current, Present
		}
		if !current.Revealed {
			return current, Unproven
		}
		current = current.childToward(key)
	}

	return nil, Absent
}

func (n *ProofNode) childToward(key Key) *ProofNode {
	if key.bitAt(n.Length) == 0 {
		return n.Left
	}

	return n.Right
}

func (n *ProofNode) Stubs() []*ProofNode {
	if n == nil {
		return nil
	}
	if !n.Revealed {
		return []*ProofNode{n}
	}

	return append(n.Left.Stubs(), n.Right.Stubs()...)
}

func (n *ProofNode) RevealedLeaves() []*ProofNode {
	if n == nil || !n.Revealed {
		return nil
	}
	if n.Leaf != nil {
		return []*ProofNode{n}
	}

	return append(n.Left.RevealedLeaves(), n.Right.RevealedLeaves()...)
}

func (p Proof) Bytes() []byte {
	return p.Root.appendBytes(nil)
}

func (n *ProofNode) appendBytes(out []byte) []byte {
	switch {
	case n == nil:
		return append(out, 0)
	case !n.Revealed:
		out = append(out, 1)
		out = binary.BigEndian.AppendUint16(out, uint16(n.Length))
		out = append(out, n.Prefix[:(n.Length+7)/8]...)
		out = append(out, n.Hash[:]...)

		return n.Summary.appendBytes(out)
	case n.Leaf != nil:
		out = append(out, 2)
		out = append(out, n.Leaf.Key[:]...)
		out = binary.BigEndian.AppendUint16(out, uint16(len(n.Leaf.Value)))

		return append(out, n.Leaf.Value...)
	default:
		out = append(out, 3)
		out = binary.BigEndian.AppendUint16(out, uint16(n.Length))
		out = append(out, n.Prefix[:(n.Length+7)/8]...)
		out = n.Left.appendBytes(out)

		return n.Right.appendBytes(out)
	}
}

func (n *ProofNode) LowestURL() URL {
	return n.Prefix.URL()
}
