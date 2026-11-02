// Package indextree holds a persistent sparse Merkle tree over fixed-length posting keys.
// The tree is a binary Patricia trie: an inner node exists only where two keys first
// differ, so its shape is unique for a set of leaves. Each node carries a summary, the
// count of its leaves and their highest impact, and every parent hashes the summaries of
// its two children apart from each other. A version is an immutable value; an update
// returns a new version that shares every unchanged node with the old one.
package indextree
