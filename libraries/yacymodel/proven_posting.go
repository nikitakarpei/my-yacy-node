package yacymodel

import "fmt"

type ProvenPosting struct {
	proof   PostingsProof
	posting RWIPosting
	root    PostingsRoot
}

func ProvenPostingFrom(posting RWIPosting, proof PostingsProof) (ProvenPosting, error) {
	postings := []RWIPosting{posting}
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

func ProvenPagePostingsJoinedFrom(provenPostings []ProvenPosting) Optional[PagePostings] {
	if len(provenPostings) == 0 {
		return None[PagePostings]()
	}
	postings := make([]RWIPosting, 0, len(provenPostings))
	mergedProof := provenPostings[0].proof
	for _, provenPosting := range provenPostings {
		postings = append(postings, provenPosting.posting)
		mergedProof = mergedProof.MergedWith(provenPosting.proof)
	}
	pagePostings, err := ProvenPagePostingsFrom(postings, mergedProof)
	if err != nil {
		return None[PagePostings]()
	}

	return Some(pagePostings)
}
