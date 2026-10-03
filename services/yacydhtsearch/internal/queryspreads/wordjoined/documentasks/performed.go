package documentasks

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

type Performed struct {
	AmountOfPeersWithANonEmptyAbstract  int
	AmountOfListedDocumentsWithMetadata int
	AmountOfListedDocumentsWithAPosting int
	AmountOfDocumentsHeldInEachAnswer   []int
}

func PerformedFrom(documentAnswers Answers) Performed {
	answers := documentAnswers.replicaAnswers()

	return Performed{
		AmountOfPeersWithANonEmptyAbstract:  amountOfPeersWithANonEmptyAbstractAmong(answers),
		AmountOfListedDocumentsWithMetadata: amountOfListedDocumentsWithMetadataAmong(answers),
		AmountOfListedDocumentsWithAPosting: amountOfListedDocumentsWithAPostingAmong(answers),
		AmountOfDocumentsHeldInEachAnswer:   amountOfDocumentsHeldPerAnswer(answers),
	}
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
