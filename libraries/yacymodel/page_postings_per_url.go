package yacymodel

func PagePostingsPerURLFrom(
	postings []RWIPosting,
) (pagePostingsPerURL []PagePostings, repeatedPostings []RWIPosting) {
	groupOfURL := make(map[URLHash]int)
	seenPostings := make(map[wordOfURL]struct{}, len(postings))
	for _, posting := range postings {
		identity := wordOfURL{word: posting.WordHash, url: posting.URLHash}
		if _, seen := seenPostings[identity]; seen {
			repeatedPostings = append(repeatedPostings, posting)

			continue
		}
		seenPostings[identity] = struct{}{}
		group, seen := groupOfURL[posting.URLHash]
		if !seen {
			group = len(pagePostingsPerURL)
			groupOfURL[posting.URLHash] = group
			pagePostingsPerURL = append(pagePostingsPerURL, PagePostings{})
		}
		pagePostingsPerURL[group].postings = append(pagePostingsPerURL[group].postings, posting)
	}

	return pagePostingsPerURL, repeatedPostings
}

type wordOfURL struct {
	word Hash
	url  URLHash
}
