package yacymodel

type PostingAlone struct {
	pagePostings PagePostings
}

func PostingAloneFrom(posting RWIPosting, proof Optional[PostingsProof]) (PostingAlone, error) {
	pagePostings, err := pagePostingsFrom([]RWIPosting{posting}, proof)
	if err != nil {
		return PostingAlone{}, err
	}

	return PostingAlone{pagePostings: pagePostings}, nil
}

func UnprovenPostingAloneFrom(posting RWIPosting) PostingAlone {
	return PostingAlone{pagePostings: PagePostings{postings: []RWIPosting{posting}}}
}

func (a PostingAlone) Posting() RWIPosting {
	return a.pagePostings.postings[0]
}

func (a PostingAlone) Proof() Optional[PostingsProof] {
	return a.pagePostings.proof
}

func (a PostingAlone) Root() Optional[PostingsRoot] {
	return a.pagePostings.Root()
}

func PagePostingsJoinedFrom(postingsAlone []PostingAlone) Optional[PagePostings] {
	postings := make([]RWIPosting, 0, len(postingsAlone))
	for _, alone := range postingsAlone {
		postings = append(postings, alone.Posting())
	}
	joined, err := pagePostingsFrom(postings, mergedProofOf(postingsAlone))
	if err != nil {
		return None[PagePostings]()
	}

	return Some(joined)
}

func mergedProofOf(postingsAlone []PostingAlone) Optional[PostingsProof] {
	proofs := make([]PostingsProof, 0, len(postingsAlone))
	for _, alone := range postingsAlone {
		proof, proven := alone.Proof().Get()
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

func pagePostingsFrom(postings []RWIPosting, proof Optional[PostingsProof]) (PagePostings, error) {
	if provenProof, proven := proof.Get(); proven {
		return ProvenPagePostingsFrom(postings, provenProof)
	}

	return UnprovenPagePostingsFrom(postings)
}
