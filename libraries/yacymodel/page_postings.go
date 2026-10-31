package yacymodel

import (
	"errors"
	"fmt"
	"slices"
)

var ErrBadPagePostings = errors.New("bad page postings")

type PagePostings struct {
	postings []RWIPosting
	proof    Optional[PostingsProof]
}

func PagePostingsFrom(
	postings []RWIPosting,
	proof Optional[PostingsProof],
) (PagePostings, error) {
	if err := failureOf(postings); err != nil {
		return PagePostings{}, err
	}
	if provenProof, proven := proof.Get(); proven && !provenProof.liesBesidePathsOf(postings) {
		return PagePostings{}, fmt.Errorf(
			"%w: a proof entry lies beside no path of the postings",
			ErrBadPagePostings,
		)
	}

	return PagePostings{postings: slices.Clone(postings), proof: proof}, nil
}

func failureOf(postings []RWIPosting) error {
	switch {
	case len(postings) == 0:
		return fmt.Errorf("%w: no postings", ErrBadPagePostings)
	case !holdsOneURL(postings):
		return fmt.Errorf("%w: postings of more than one url", ErrBadPagePostings)
	case !holdsEachWordOnce(postings):
		return fmt.Errorf("%w: a word more than once", ErrBadPagePostings)
	default:
		return nil
	}
}

func holdsOneURL(postings []RWIPosting) bool {
	for _, posting := range postings {
		if posting.URLHash != postings[0].URLHash {
			return false
		}
	}

	return true
}

func holdsEachWordOnce(postings []RWIPosting) bool {
	words := make(map[[HashByteLength]byte]struct{}, len(postings))
	for _, posting := range postings {
		words[posting.WordHash.raw] = struct{}{}
	}

	return len(words) == len(postings)
}

func (p PagePostings) Postings() []RWIPosting {
	return slices.Clone(p.postings)
}

func (p PagePostings) AmountOfPostings() int {
	return len(p.postings)
}

func (p PagePostings) Proof() Optional[PostingsProof] {
	return p.proof
}

func (p PagePostings) URL() URLHash {
	if len(p.postings) == 0 {
		return URLHash{}
	}

	return p.postings[0].URLHash
}

func (p PagePostings) Root() Optional[PostingsRoot] {
	proof, proven := p.proof.Get()
	if !proven {
		return None[PostingsRoot]()
	}

	return Some(PostingsRoot{
		digest:            p.treeWith(proof).hash(),
		partitionExponent: proof.partitionExponent,
	})
}

func (p PagePostings) treeWith(proof PostingsProof) postingsSubtree {
	return postingsSubtree{
		leaves:  leavesOf(p.postings, proof.partitionExponent),
		entries: proof.entries,
	}
}

func (p PagePostings) NarrowedTo(keptPostings []RWIPosting) Optional[PagePostings] {
	if !p.holdsEach(keptPostings) {
		return None[PagePostings]()
	}
	narrowed, err := PagePostingsFrom(keptPostings, p.proofNarrowedTo(keptPostings))
	if err != nil {
		return None[PagePostings]()
	}

	return Some(narrowed)
}

func (p PagePostings) holdsEach(postings []RWIPosting) bool {
	held := make(map[RWIPosting]struct{}, len(p.postings))
	for _, posting := range p.postings {
		held[posting] = struct{}{}
	}
	for _, posting := range postings {
		if _, found := held[posting]; !found {
			return false
		}
	}

	return true
}

func (p PagePostings) proofNarrowedTo(keptPostings []RWIPosting) Optional[PostingsProof] {
	proof, proven := p.proof.Get()
	if !proven {
		return None[PostingsProof]()
	}

	return Some(PostingsProof{
		entries: p.treeWith(proof).
			entriesAround(leafPositionsOf(keptPostings, proof.partitionExponent)),
		partitionExponent: proof.partitionExponent,
	})
}
