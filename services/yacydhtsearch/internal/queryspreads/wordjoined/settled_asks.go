package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"

type settledAsks []wordpartitionasks.SettledAsk

func (asks settledAsks) answers() []wordpartitionasks.ReplicaAnswer {
	var answers []wordpartitionasks.ReplicaAnswer
	for _, settledAsk := range asks {
		answers = append(answers, settledAsk.Answers...)
	}

	return answers
}
