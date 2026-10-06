// Package documentjoin joins the documents of one query as the word partitions
// answer. A document joins once every query word listed it, a compound word
// listing it for each of its part words. The join tells its observer the
// documents that joined on each listing.
package documentjoin

import (
	"maps"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type JoinObserver interface {
	DocumentsJoined(documents yacymodel.URLHashes)
}

type JoinObservers []JoinObserver

func (observers JoinObservers) DocumentsJoined(documents yacymodel.URLHashes) {
	for _, observer := range observers {
		observer.DocumentsJoined(documents)
	}
}

type Joiner struct {
	observer                    JoinObserver
	query                       searchquery.Query
	listedDocumentsPerQueryWord listedDocumentsPerQueryWord
	joinedDocuments             yacymodel.URLHashes
}

func JoinerOf(query searchquery.Query, observer JoinObserver) *Joiner {
	return &Joiner{
		observer:                    observer,
		query:                       query,
		listedDocumentsPerQueryWord: noListedDocumentsPerQueryWord(query.WordHashes()),
		joinedDocuments:             yacymodel.URLHashes{},
	}
}

func (joiner *Joiner) ListUnder(word yacymodel.Hash, documents yacymodel.URLHashes) {
	joiner.listUnderTheQueryWordsOf(word, documents)
	joinedDocuments := joiner.newlyJoinedAmong(documents)
	if len(joinedDocuments) == 0 {
		return
	}
	joiner.keep(joinedDocuments)
	joiner.observer.DocumentsJoined(joinedDocuments)
}

func (joiner *Joiner) listUnderTheQueryWordsOf(word yacymodel.Hash, documents yacymodel.URLHashes) {
	if slices.Contains(joiner.query.WordHashes(), word) {
		joiner.listedDocumentsPerQueryWord.add(word, documents)
	}
	for _, compoundWord := range joiner.query.CompoundWords {
		if compoundWord.Hash() != word {
			continue
		}
		for _, partWord := range compoundWord.PartHashes() {
			joiner.listedDocumentsPerQueryWord.add(partWord, documents)
		}
	}
}

func (joiner *Joiner) newlyJoinedAmong(documents yacymodel.URLHashes) yacymodel.URLHashes {
	joinedDocuments := yacymodel.URLHashes{}
	for document := range documents {
		if joiner.joinedDocuments.Contains(document) ||
			!joiner.listedDocumentsPerQueryWord.listedForEveryQueryWord(document) {
			continue
		}
		joinedDocuments.Add(document)
	}

	return joinedDocuments
}

func (joiner *Joiner) keep(joinedDocuments yacymodel.URLHashes) {
	maps.Copy(joiner.joinedDocuments, joinedDocuments)
}

func (joiner *Joiner) JoinedDocuments() yacymodel.URLHashes {
	return maps.Clone(joiner.joinedDocuments)
}
