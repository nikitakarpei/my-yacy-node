package searchresult

type Ranking struct {
	Items []Item `json:"items"`
}

func (r Ranking) PageFrom(skippedItems, wantedItems int) Page {
	if skippedItems < 0 || skippedItems >= len(r.Items) || wantedItems <= 0 {
		return Page{}
	}

	return Page{
		Items: r.Items[skippedItems:min(skippedItems+wantedItems, len(r.Items))],
	}
}
