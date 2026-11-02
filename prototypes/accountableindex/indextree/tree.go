package indextree

import (
	"crypto/sha256"
	"encoding/binary"
)

type Hash [sha256.Size]byte

type Leaf struct {
	Value []byte
	Key   Key
}

type Changes struct {
	Stored  []Leaf
	Removed []Key
}

type Tree struct {
	root      *node
	summaryOf SummaryRule
	version   uint64
}

func EmptyTree(summaryOf SummaryRule) Tree {
	return Tree{summaryOf: summaryOf}
}

func (t Tree) Root() Hash {
	return t.root.hashOrZero()
}

func (t Tree) Summary() Summary {
	return t.root.summaryOrEmpty()
}

func (t Tree) Version() uint64 {
	return t.version
}

func (t Tree) Updated(changes Changes) (updated Tree, amountOfWrittenNodes int) {
	version := t.version + 1
	writer := nodeWriter{summaryOf: t.summaryOf, version: version}
	root := t.root
	for _, leaf := range changes.Stored {
		root = writer.stored(root, leaf)
	}
	for _, key := range changes.Removed {
		root = writer.removed(root, key)
	}

	return Tree{root: root, summaryOf: t.summaryOf, version: version}, root.amountOfNodesOfVersion(version)
}

func (t Tree) Value(key Key) ([]byte, bool) {
	current := t.root
	for current != nil && current.leaf == nil {
		if !key.sharesPrefixWith(current.prefix, current.split) {
			return nil, false
		}
		current = current.childToward(key)
	}
	if current == nil || current.leaf.Key != key {
		return nil, false
	}

	return current.leaf.Value, true
}

type node struct {
	leaf    *Leaf
	left    *node
	right   *node
	summary Summary
	version uint64
	split   int
	hash    Hash
	prefix  Key
}

func (n *node) hashOrZero() Hash {
	if n == nil {
		return Hash{}
	}

	return n.hash
}

func (n *node) summaryOrEmpty() Summary {
	if n == nil {
		return Summary{}
	}

	return n.summary
}

func (n *node) childToward(key Key) *node {
	if key.bitAt(n.split) == 0 {
		return n.left
	}

	return n.right
}

func (n *node) amountOfNodesOfVersion(version uint64) int {
	if n == nil || n.version != version {
		return 0
	}

	return 1 + n.left.amountOfNodesOfVersion(version) + n.right.amountOfNodesOfVersion(version)
}

func (n *node) keyRange() (prefix Key, length int) {
	if n.leaf != nil {
		return n.leaf.Key, keyBits
	}

	return n.prefix, n.split
}

type nodeWriter struct {
	summaryOf SummaryRule
	version   uint64
}

func (w nodeWriter) stored(current *node, leaf Leaf) *node {
	if current == nil {
		return w.leafNode(leaf)
	}
	prefix, length := current.keyRange()
	if !leaf.Key.sharesPrefixWith(prefix, length) || (current.leaf != nil && current.leaf.Key != leaf.Key) {
		return w.innerNode(current, w.leafNode(leaf))
	}
	if current.leaf != nil {
		return w.leafNode(leaf)
	}
	if leaf.Key.bitAt(current.split) == 0 {
		return w.innerNodeOf(current.split, current.prefix, w.stored(current.left, leaf), current.right)
	}

	return w.innerNodeOf(current.split, current.prefix, current.left, w.stored(current.right, leaf))
}

func (w nodeWriter) removed(current *node, key Key) *node {
	if current == nil {
		return nil
	}
	if current.leaf != nil {
		if current.leaf.Key == key {
			return nil
		}

		return current
	}
	if !key.sharesPrefixWith(current.prefix, current.split) {
		return current
	}
	left, right := current.left, current.right
	if key.bitAt(current.split) == 0 {
		left = w.removed(left, key)
	} else {
		right = w.removed(right, key)
	}
	switch {
	case left == current.left && right == current.right:
		return current
	case left == nil:
		return right
	case right == nil:
		return left
	default:
		return w.innerNodeOf(current.split, current.prefix, left, right)
	}
}

func (w nodeWriter) innerNode(existing, added *node) *node {
	existingPrefix, _ := existing.keyRange()
	addedPrefix, _ := added.keyRange()
	split := firstDifferingBitOf(existingPrefix, addedPrefix)
	if addedPrefix.bitAt(split) == 0 {
		return w.innerNodeOf(split, addedPrefix.prefixOfLength(split), added, existing)
	}

	return w.innerNodeOf(split, addedPrefix.prefixOfLength(split), existing, added)
}

func (w nodeWriter) leafNode(leaf Leaf) *node {
	return &node{
		leaf:    &leaf,
		summary: w.summaryOf(leaf.Value),
		version: w.version,
		hash:    leafHashOf(leaf.Key, leaf.Value),
	}
}

func (w nodeWriter) innerNodeOf(split int, prefix Key, left, right *node) *node {
	return &node{
		left:    left,
		right:   right,
		summary: left.summary.combinedWith(right.summary),
		version: w.version,
		split:   split,
		prefix:  prefix,
		hash:    innerHashOf(split, prefix, left.hash, left.summary, right.hash, right.summary),
	}
}

func leafHashOf(key Key, value []byte) Hash {
	valueHash := sha256.Sum256(value)
	input := make([]byte, 0, 1+KeyLength+sha256.Size)
	input = append(input, 0)
	input = append(input, key[:]...)
	input = append(input, valueHash[:]...)

	return sha256.Sum256(input)
}

func innerHashOf(split int, prefix Key, leftHash Hash, leftSummary Summary, rightHash Hash, rightSummary Summary) Hash {
	input := make([]byte, 0, 3+KeyLength+2*(sha256.Size+summaryLength))
	input = append(input, 1)
	input = binary.BigEndian.AppendUint16(input, uint16(split))
	input = append(input, prefix[:]...)
	input = append(input, leftHash[:]...)
	input = leftSummary.appendBytes(input)
	input = append(input, rightHash[:]...)
	input = rightSummary.appendBytes(input)

	return sha256.Sum256(input)
}
