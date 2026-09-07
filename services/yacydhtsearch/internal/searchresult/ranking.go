package searchresult

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type Ranking struct {
	Items []Item `json:"items"`
}

func RankingFrom(answers [][]Item, ceiling int) Ranking {
	if ceiling <= 0 {
		return Ranking{}
	}

	items := make([]Item, 0, ceiling)
	takenHashes := map[yacymodel.URLHash]struct{}{}
	for round := 0; round < amountOfRounds(answers) && len(items) < ceiling; round++ {
		for _, answer := range answers {
			if round >= len(answer) || len(items) == ceiling {
				continue
			}
			item := answer[round]
			if _, seen := takenHashes[item.Hash]; seen {
				continue
			}
			takenHashes[item.Hash] = struct{}{}
			items = append(items, item)
		}
	}

	return Ranking{Items: items}
}

func (r Ranking) PageFrom(skippedItems, wantedItems int) Page {
	if skippedItems < 0 || skippedItems >= len(r.Items) || wantedItems <= 0 {
		return Page{}
	}

	return Page{
		Items: r.Items[skippedItems:min(skippedItems+wantedItems, len(r.Items))],
	}
}

func amountOfRounds(answers [][]Item) int {
	var rounds int
	for _, answer := range answers {
		rounds = max(rounds, len(answer))
	}

	return rounds
}
