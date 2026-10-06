package documentasks

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

type Performed struct {
	AmountOfPeersWithANonEmptyAbstract  int
	AmountOfListedDocumentsWithMetadata int
	AmountOfListedDocumentsWithAPosting int
	AmountOfDocumentsHeldInEachAnswer   []int
	AmountOfCompoundWordsAnswered       int
}

func performedFrom(
	answered []wordpartitionasks.SettledAsk,
	compoundWords []searchquery.CompoundWord,
) Performed {
	answers := replicaAnswersIn(answered)

	return Performed{
		AmountOfPeersWithANonEmptyAbstract:  amountOfPeersWithANonEmptyAbstractAmong(answers),
		AmountOfListedDocumentsWithMetadata: amountOfListedDocumentsWithMetadataAmong(answers),
		AmountOfListedDocumentsWithAPosting: amountOfListedDocumentsWithAPostingAmong(answers),
		AmountOfDocumentsHeldInEachAnswer:   amountOfDocumentsHeldPerAnswer(answers),
		AmountOfCompoundWordsAnswered: amountOfCompoundWordsAnsweredAmong(
			compoundWords,
			answered,
		),
	}
}

func replicaAnswersIn(answered []wordpartitionasks.SettledAsk) []wordpartitionasks.ReplicaAnswer {
	var answers []wordpartitionasks.ReplicaAnswer
	for _, settledAsk := range answered {
		answers = append(answers, settledAsk.Answers...)
	}

	return answers
}

func amountOfPeersWithANonEmptyAbstractAmong(answers []wordpartitionasks.ReplicaAnswer) int {
	peers := map[peerdirectory.AskablePeer]struct{}{}
	for _, answer := range answers {
		if len(answer.ListedDocuments) == 0 {
			continue
		}
		peers[answer.Replica] = struct{}{}
	}

	return len(peers)
}

func amountOfListedDocumentsWithMetadataAmong(answers []wordpartitionasks.ReplicaAnswer) int {
	amount := 0
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if !listedDocument.Metadata.Present() {
				continue
			}
			amount++
		}
	}

	return amount
}

func amountOfListedDocumentsWithAPostingAmong(answers []wordpartitionasks.ReplicaAnswer) int {
	amount := 0
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if !listedDocument.Posting.Present() {
				continue
			}
			amount++
		}
	}

	return amount
}

func amountOfDocumentsHeldPerAnswer(answers []wordpartitionasks.ReplicaAnswer) []int {
	amountOfDocumentsHeldInEachAnswer := make([]int, 0, len(answers))
	for _, answer := range answers {
		amountOfDocumentsHeldForTheWord, counted := answer.AmountOfDocumentsHeld.Get()
		if !counted {
			continue
		}
		amountOfDocumentsHeldInEachAnswer = append(
			amountOfDocumentsHeldInEachAnswer, amountOfDocumentsHeldForTheWord,
		)
	}

	return amountOfDocumentsHeldInEachAnswer
}

func amountOfCompoundWordsAnsweredAmong(
	compoundWords []searchquery.CompoundWord,
	answered []wordpartitionasks.SettledAsk,
) int {
	amount := 0
	for _, compoundWord := range compoundWords {
		if slices.ContainsFunc(answered, func(settledAsk wordpartitionasks.SettledAsk) bool {
			return settledAsk.Word == compoundWord.Hash()
		}) {
			amount++
		}
	}

	return amount
}
