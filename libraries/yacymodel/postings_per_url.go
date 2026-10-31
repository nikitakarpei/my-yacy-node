package yacymodel

func PostingsPerURLFrom(
	postings []RWIPosting,
) (postingsPerURL [][]RWIPosting, repeatedPostings []RWIPosting) {
	groupOfURL := make(map[URLHash]int)
	seenPostings := make(map[postingIdentity]struct{}, len(postings))
	for _, posting := range postings {
		identity := postingIdentity{word: posting.WordHash, url: posting.URLHash}
		if _, seen := seenPostings[identity]; seen {
			repeatedPostings = append(repeatedPostings, posting)

			continue
		}
		seenPostings[identity] = struct{}{}
		group, seen := groupOfURL[posting.URLHash]
		if !seen {
			group = len(postingsPerURL)
			groupOfURL[posting.URLHash] = group
			postingsPerURL = append(postingsPerURL, nil)
		}
		postingsPerURL[group] = append(postingsPerURL[group], posting)
	}

	return postingsPerURL, repeatedPostings
}

type postingIdentity struct {
	word Hash
	url  URLHash
}
