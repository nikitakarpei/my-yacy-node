package yacymodel

type PostingAlone struct {
	proof   Optional[PostingsProof]
	posting RWIPosting
	root    Optional[PostingsRoot]
}

func ProvenPostingAloneFrom(posting RWIPosting, proof PostingsProof) (PostingAlone, error) {
	pagePostings, err := ProvenPagePostingsFrom([]RWIPosting{posting}, proof)
	if err != nil {
		return PostingAlone{}, err
	}

	return PostingAlone{posting: posting, proof: Some(proof), root: pagePostings.Root()}, nil
}

func UnprovenPostingAloneFrom(posting RWIPosting) PostingAlone {
	return PostingAlone{posting: posting}
}

func (a PostingAlone) Posting() RWIPosting {
	return a.posting
}

func (a PostingAlone) Proof() Optional[PostingsProof] {
	return a.proof
}

func (a PostingAlone) Root() Optional[PostingsRoot] {
	return a.root
}

func PagePostingsJoinedFrom(postingsAlone []PostingAlone) Optional[PagePostings] {
	postings := make([]RWIPosting, 0, len(postingsAlone))
	for _, alone := range postingsAlone {
		postings = append(postings, alone.posting)
	}
	proof, proven := mergedProofOf(postingsAlone).Get()
	if !proven {
		return presentUnlessRefused(UnprovenPagePostingsFrom(postings))
	}

	return presentUnlessRefused(ProvenPagePostingsFrom(postings, proof))
}

func mergedProofOf(postingsAlone []PostingAlone) Optional[PostingsProof] {
	proofs := make([]PostingsProof, 0, len(postingsAlone))
	for _, alone := range postingsAlone {
		proof, proven := alone.proof.Get()
		if !proven {
			return None[PostingsProof]()
		}
		proofs = append(proofs, proof)
	}
	if len(proofs) == 0 {
		return None[PostingsProof]()
	}
	merged := proofs[0]
	for _, proof := range proofs[1:] {
		merged = merged.MergedWith(proof)
	}

	return Some(merged)
}

func presentUnlessRefused(pagePostings PagePostings, err error) Optional[PagePostings] {
	if err != nil {
		return None[PagePostings]()
	}

	return Some(pagePostings)
}
