package yacymodel

import "fmt"

type ProvenPosting struct {
	proof   PostingsProof
	posting RWIPosting
	root    PostingsRoot
}

func ProvenPostingFrom(posting RWIPosting, proof PostingsProof) (ProvenPosting, error) {
	postings := []RWIPosting{posting}
	if err := failureOf(postings); err != nil {
		return ProvenPosting{}, err
	}
	if !proof.liesBesidePathsOf(postings) {
		return ProvenPosting{}, fmt.Errorf(
			"%w: a proof entry lies beside no path of the posting",
			ErrBadPostingsProof,
		)
	}

	return ProvenPosting{posting: posting, proof: proof, root: proof.rootOver(postings)}, nil
}

func (p ProvenPosting) Posting() RWIPosting {
	return p.posting
}

func (p ProvenPosting) Proof() PostingsProof {
	return p.proof
}

func (p ProvenPosting) Root() PostingsRoot {
	return p.root
}

func PagePostingsJoinedFrom(provenPostings []ProvenPosting) (PagePostings, error) {
	postings := postingsOfEach(provenPostings)
	if err := failureOf(postings); err != nil {
		return PagePostings{}, err
	}
	if !holdsOneRoot(provenPostings) {
		return PagePostings{}, fmt.Errorf(
			"%w: proven postings of more than one root",
			ErrBadPagePostings,
		)
	}
	mergedProof := provenPostings[0].proof
	for _, provenPosting := range provenPostings[1:] {
		mergedProof = mergedProof.MergedWith(provenPosting.proof)
	}

	return PagePostings{
		postings: postings,
		proof:    Some(mergedProof.narrowedFrom(postings, postings)),
	}, nil
}

func postingsOfEach(provenPostings []ProvenPosting) []RWIPosting {
	postings := make([]RWIPosting, 0, len(provenPostings))
	for _, provenPosting := range provenPostings {
		postings = append(postings, provenPosting.posting)
	}

	return postings
}

func holdsOneRoot(provenPostings []ProvenPosting) bool {
	for _, provenPosting := range provenPostings[1:] {
		if provenPosting.root != provenPostings[0].root {
			return false
		}
	}

	return true
}
