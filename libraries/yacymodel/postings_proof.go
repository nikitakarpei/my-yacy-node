package yacymodel

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/internal/postingstree"
)

const postingsProofFormat byte = 1

var ErrBadPostingsProof = errors.New("bad postings proof")

type PostingsProof struct {
	entries           []postingstree.Node
	partitionExponent uint8
}

func EmptyPostingsProof(partitions DHTRingPartitions) PostingsProof {
	return PostingsProof{partitionExponent: partitions.exponentByte()}
}

func (p PostingsProof) MergedWith(other PostingsProof) PostingsProof {
	return p.withEntries(postingstree.UnionOf(p.entries, other.entries))
}

func (p PostingsProof) withEntries(entries []postingstree.Node) PostingsProof {
	return PostingsProof{entries: entries, partitionExponent: p.partitionExponent}
}

func (p PostingsProof) NarrowedTo(
	remainingWords []Hash,
	purgedPostings []RWIPosting,
) PostingsProof {
	remainingPositions := leafPositionsOfWords(remainingWords, p.partitionExponent)
	knownSubtrees := postingstree.SubtreeOf(nil, postingstree.OutermostNodesOf(
		postingstree.EntriesOutside(p.entries, remainingPositions),
		leavesOf(purgedPostings, p.partitionExponent),
	))

	return p.withEntries(knownSubtrees.EntriesAround(remainingPositions))
}

func (p PostingsProof) liesBesidePathsOf(postings []RWIPosting) bool {
	return postingstree.LieBesidePathsOf(
		p.entries,
		leafPositionsOfPostings(postings, p.partitionExponent),
	)
}

func (p PostingsProof) rootOver(postings []RWIPosting) PostingsRoot {
	return p.rootOf(p.subtreeOver(postings))
}

func (p PostingsProof) rootOf(subtree postingstree.Subtree) PostingsRoot {
	return PostingsRoot{digest: subtree.Hash(), partitionExponent: p.partitionExponent}
}

func (p PostingsProof) subtreeOver(postings []RWIPosting) postingstree.Subtree {
	return postingstree.SubtreeOf(leavesOf(postings, p.partitionExponent), p.entries)
}

func (p PostingsProof) narrowedFrom(postings, keptPostings []RWIPosting) PostingsProof {
	return p.withEntries(p.subtreeOver(postings).EntriesAround(
		leafPositionsOfPostings(keptPostings, p.partitionExponent),
	))
}

func (p PostingsProof) provenPostingsFrom(postings []RWIPosting) []ProvenPosting {
	subtree := p.subtreeOver(postings)
	root := p.rootOf(subtree)
	eachEntries := subtree.EntriesAroundEach(leafPositionsOfPostings(postings, p.partitionExponent))
	provenPostings := make([]ProvenPosting, 0, len(eachEntries))
	for index, entries := range eachEntries {
		provenPostings = append(provenPostings, ProvenPosting{
			posting: postings[index],
			proof:   p.withEntries(entries),
			root:    root,
		})
	}

	return provenPostings
}

func (p PostingsProof) Bytes() []byte {
	return postingstree.AppendEntries([]byte{postingsProofFormat, p.partitionExponent}, p.entries)
}

func ParsePostingsProofBytes(raw []byte) (PostingsProof, error) {
	if len(raw) < 2 || raw[0] != postingsProofFormat {
		return PostingsProof{}, fmt.Errorf(
			"%w: no format %d header",
			ErrBadPostingsProof,
			postingsProofFormat,
		)
	}
	if _, err := DHTRingPartitionsFromExponent(uint(raw[1])); err != nil {
		return PostingsProof{}, fmt.Errorf("%w: %w", ErrBadPostingsProof, err)
	}
	entries, err := postingstree.EntriesFrom(raw[2:])
	if err != nil {
		return PostingsProof{}, fmt.Errorf("%w: %w", ErrBadPostingsProof, err)
	}

	return PostingsProof{entries: entries, partitionExponent: raw[1]}, nil
}
