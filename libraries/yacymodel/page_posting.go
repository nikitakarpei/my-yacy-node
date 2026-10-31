package yacymodel

import "fmt"

type PagePosting struct {
	proof   Optional[PostingsProof]
	posting RWIPosting
	root    Optional[PostingsRoot]
}

func UnprovenPagePostingFrom(posting RWIPosting) (PagePosting, error) {
	if err := failureOf([]RWIPosting{posting}); err != nil {
		return PagePosting{}, err
	}

	return PagePosting{posting: posting}, nil
}

func ProvenPagePostingFrom(posting RWIPosting, proof PostingsProof) (PagePosting, error) {
	postings := []RWIPosting{posting}
	if err := failureOf(postings); err != nil {
		return PagePosting{}, err
	}
	if !proof.liesBesidePathsOf(postings) {
		return PagePosting{}, fmt.Errorf(
			"%w: a proof entry lies beside no path of the posting",
			ErrBadPostingsProof,
		)
	}

	return PagePosting{
		posting: posting,
		proof:   Some(proof),
		root:    Some(proof.rootOver(postings)),
	}, nil
}

func (p PagePosting) Posting() RWIPosting {
	return p.posting
}

func (p PagePosting) Word() Hash {
	return p.posting.WordHash
}

func (p PagePosting) URL() URLHash {
	return p.posting.URLHash
}

func (p PagePosting) Language() Language {
	return p.posting.Language
}

func (p PagePosting) Proof() Optional[PostingsProof] {
	return p.proof
}

func (p PagePosting) Root() Optional[PostingsRoot] {
	return p.root
}

func PagePostingsJoinedFrom(pagePostings []PagePosting) (PagePostings, error) {
	postings := postingsOfEach(pagePostings)
	if err := failureOf(postings); err != nil {
		return PagePostings{}, err
	}
	if !holdsOneRoot(pagePostings) {
		return PagePostings{}, fmt.Errorf(
			"%w: page postings of more than one root",
			ErrBadPagePostings,
		)
	}

	return PagePostings{
		postings: postings,
		proof:    proofJoinedFrom(pagePostings, postings),
		root:     pagePostings[0].root,
	}, nil
}

func postingsOfEach(pagePostings []PagePosting) []RWIPosting {
	postings := make([]RWIPosting, 0, len(pagePostings))
	for _, pagePosting := range pagePostings {
		postings = append(postings, pagePosting.posting)
	}

	return postings
}

func holdsOneRoot(pagePostings []PagePosting) bool {
	for _, pagePosting := range pagePostings[1:] {
		if pagePosting.root != pagePostings[0].root {
			return false
		}
	}

	return true
}

func proofJoinedFrom(
	pagePostings []PagePosting,
	postings []RWIPosting,
) Optional[PostingsProof] {
	mergedProof, proven := pagePostings[0].proof.Get()
	if !proven {
		return None[PostingsProof]()
	}
	for _, pagePosting := range pagePostings[1:] {
		proof, _ := pagePosting.proof.Get()
		mergedProof = mergedProof.MergedWith(proof)
	}

	return Some(mergedProof.narrowedFrom(postings, postings))
}
