package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type discoveryAnswers struct {
	queryWords            []yacymodel.Hash
	compoundWords         []searchquery.CompoundWord
	partitions            yacymodel.DHTRingPartitions
	wordsAcrossReplicas   map[yacymodel.Hash]wordAcrossReplicas
	holdersPerDocument    holdersPerDocument
	queryWordsPerDocument queryWordsPerDocument
	documentsThePeersSent *queryfindings.DocumentsThePeersSent
}

func discoveryAnswersFor(
	query searchquery.Query,
	partitions yacymodel.DHTRingPartitions,
) *discoveryAnswers {
	return &discoveryAnswers{
		queryWords:            query.WordHashes(),
		compoundWords:         query.CompoundWords,
		partitions:            partitions,
		wordsAcrossReplicas:   map[yacymodel.Hash]wordAcrossReplicas{},
		holdersPerDocument:    holdersPerDocument{},
		queryWordsPerDocument: queryWordsPerDocument{},
		documentsThePeersSent: queryfindings.EmptyDocumentsThePeersSent(),
	}
}

func (answers *discoveryAnswers) add(settledAsk wordpartitionasks.SettledAsk) {
	answers.wordsAcrossReplicas[settledAsk.Word] = answers.wordAcrossReplicasOf(settledAsk.Word).
		withAnswersOf(settledAsk)
	answers.holdersPerDocument.addHoldersIn(settledAsk.Answers)
	answers.queryWordsPerDocument.addListingsIn(
		settledAsk.Answers, answers.queryWordsOf(settledAsk.Word),
	)
	answers.keepTheDocumentsThePeersMatchedIn(settledAsk)
}

func (answers *discoveryAnswers) wordAcrossReplicasOf(word yacymodel.Hash) wordAcrossReplicas {
	answeredWord, answered := answers.wordsAcrossReplicas[word]
	if !answered {
		return wordAcrossReplicas{
			hash:                word,
			answersPerPartition: make([][]wordpartitionasks.ReplicaAnswer, answers.partitions),
		}
	}

	return answeredWord
}

func (answers *discoveryAnswers) queryWordsOf(word yacymodel.Hash) []yacymodel.Hash {
	place := slices.IndexFunc(
		answers.compoundWords,
		func(compoundWord searchquery.CompoundWord) bool {
			return compoundWord.Hash() == word
		},
	)
	if place < 0 {
		return []yacymodel.Hash{word}
	}

	return answers.compoundWords[place].PartHashes()
}

func (answers *discoveryAnswers) keepTheDocumentsThePeersMatchedIn(
	settledAsk wordpartitionasks.SettledAsk,
) {
	for _, answer := range settledAsk.Answers {
		for _, listedDocument := range answer.ListedDocuments {
			metadata, matched := listedDocument.Metadata.Get()
			if !matched {
				continue
			}
			answers.documentsThePeersSent.KeepDocumentThePeerMatched(
				answer.Holder.Hash,
				settledAsk.Word,
				metadata,
				listedDocument.Posting,
			)
		}
	}
}

func (answers *discoveryAnswers) queryWordsAcrossReplicas() []wordAcrossReplicas {
	queryWordsAcrossReplicas := make([]wordAcrossReplicas, 0, len(answers.queryWords))
	for _, word := range answers.queryWords {
		queryWordsAcrossReplicas = append(
			queryWordsAcrossReplicas,
			answers.wordAcrossReplicasOf(word),
		)
	}

	return queryWordsAcrossReplicas
}

func (answers *discoveryAnswers) queryWordsFewestDocumentsFirst() []wordAcrossReplicas {
	queryWordsAcrossReplicas := answers.queryWordsAcrossReplicas()
	slices.SortStableFunc(queryWordsAcrossReplicas, fewestDocumentsFirst)

	return queryWordsAcrossReplicas
}

func (answers *discoveryAnswers) compoundWordsAcrossReplicas() []compoundWordAcrossReplicas {
	var compoundWordsAcrossReplicas []compoundWordAcrossReplicas
	for _, compoundWord := range answers.compoundWords {
		answeredWord, answered := answers.wordsAcrossReplicas[compoundWord.Hash()]
		if !answered {
			continue
		}
		compoundWordsAcrossReplicas = append(
			compoundWordsAcrossReplicas,
			compoundWordAcrossReplicas{
				CompoundWord:       compoundWord,
				wordAcrossReplicas: answeredWord,
			},
		)
	}

	return compoundWordsAcrossReplicas
}

func (answers *discoveryAnswers) amountOfDocumentsHeldPerQueryWord() map[yacymodel.Hash]int {
	amountOfDocumentsHeldPerQueryWord := make(map[yacymodel.Hash]int, len(answers.queryWords))
	for _, queryWord := range answers.queryWordsAcrossReplicas() {
		amountOfDocumentsHeld, counted := queryWord.estimatedAmountOfDocumentsHeld().Get()
		if !counted {
			continue
		}
		amountOfDocumentsHeldPerQueryWord[queryWord.hash] = amountOfDocumentsHeld
	}

	return amountOfDocumentsHeldPerQueryWord
}

func (answers *discoveryAnswers) documentsOfWordsIn(
	words []yacymodel.Hash,
	partition uint,
) distinctDocuments {
	documents := distinctDocuments{}
	for _, word := range words {
		for document := range answers.wordAcrossReplicasOf(word).documents() {
			if answers.partitions.PartitionOf(document) != partition {
				continue
			}
			documents.add(document)
		}
	}

	return documents
}

func (answers *discoveryAnswers) mostHeldFirst(documents distinctDocuments) []yacymodel.URLHash {
	return answers.holdersPerDocument.mostHeldFirst(documents)
}

func (answers *discoveryAnswers) joinedDocuments() distinctDocuments {
	return answers.queryWordsPerDocument.documentsWithEvery(answers.queryWords)
}

func (answers *discoveryAnswers) joinedDocumentsThePeersSent() *queryfindings.DocumentsThePeersSent {
	return answers.documentsThePeersSent.Among(answers.joinedDocuments())
}

func (answers *discoveryAnswers) foundDocuments() []queryfindings.FoundDocument {
	return answers.joinedDocumentsThePeersSent().FoundDocuments()
}

func (answers *discoveryAnswers) joinedDocumentsWithoutMetadata() distinctDocuments {
	documentsWithoutMetadata := distinctDocuments{}
	for document := range answers.joinedDocuments() {
		if answers.documentsThePeersSent.Contains(document) {
			continue
		}
		documentsWithoutMetadata.add(document)
	}

	return documentsWithoutMetadata
}

func (answers *discoveryAnswers) everyAnswer() []wordpartitionasks.ReplicaAnswer {
	var everyAnswer []wordpartitionasks.ReplicaAnswer
	for _, queryWord := range answers.queryWordsAcrossReplicas() {
		everyAnswer = append(everyAnswer, queryWord.everyAnswer()...)
	}
	for _, compoundWord := range answers.compoundWordsAcrossReplicas() {
		everyAnswer = append(everyAnswer, compoundWord.everyAnswer()...)
	}

	return everyAnswer
}
