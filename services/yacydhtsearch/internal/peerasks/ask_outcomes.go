package peerasks

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type AskOutcome[Ask any, Answered any] struct {
	Ask    Ask
	Put    bool
	Answer yacymodel.Optional[Answered]
}

type AskOutcomes[Ask any, Answered any] []AskOutcome[Ask, Answered]

func (outcomes AskOutcomes[Ask, Answered]) AsksPut() []Ask {
	asksPut := make([]Ask, 0, len(outcomes))
	for _, outcome := range outcomes {
		if !outcome.Put {
			continue
		}
		asksPut = append(asksPut, outcome.Ask)
	}

	return asksPut
}

func (outcomes AskOutcomes[Ask, Answered]) AnsweredAsks() []Answered {
	answeredAsks := make([]Answered, 0, len(outcomes))
	for _, outcome := range outcomes {
		answeredAsk, answered := outcome.Answer.Get()
		if !answered {
			continue
		}
		answeredAsks = append(answeredAsks, answeredAsk)
	}

	return answeredAsks
}
