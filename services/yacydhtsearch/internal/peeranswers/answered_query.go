// Package peeranswers holds what the peers answered for one whole query: the
// items of each answer in the order the peer put them, the items in no order,
// how many documents the peers hold per query word, and the one item of each
// answered document. The text of a document, once a node read its page,
// replaces what the peers counted for it on every item of the document.
package peeranswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type AnsweredQuery struct {
	ItemsInTheOrderOfEachAnswer [][]AnsweredItem
	ItemsInNoOrder              []AnsweredItem
	DocumentsHeldPerQueryWord   map[yacymodel.Hash]int
}

func (a AnsweredQuery) CarryingTheTextOfEachDocument(
	textPerDocument map[yacymodel.URLHash]documenttext.DocumentText,
) AnsweredQuery {
	if len(textPerDocument) == 0 {
		return a
	}

	itemsInTheOrderOfEachAnswer := make(
		[][]AnsweredItem, 0, len(a.ItemsInTheOrderOfEachAnswer),
	)
	for _, itemsOfOneAnswer := range a.ItemsInTheOrderOfEachAnswer {
		itemsInTheOrderOfEachAnswer = append(
			itemsInTheOrderOfEachAnswer,
			itemsCarryingTheTextOfTheirDocument(itemsOfOneAnswer, textPerDocument),
		)
	}

	return AnsweredQuery{
		ItemsInTheOrderOfEachAnswer: itemsInTheOrderOfEachAnswer,
		ItemsInNoOrder: itemsCarryingTheTextOfTheirDocument(
			a.ItemsInNoOrder, textPerDocument,
		),
		DocumentsHeldPerQueryWord: a.DocumentsHeldPerQueryWord,
	}
}

func itemsCarryingTheTextOfTheirDocument(
	items []AnsweredItem,
	textPerDocument map[yacymodel.URLHash]documenttext.DocumentText,
) []AnsweredItem {
	carryingItems := make([]AnsweredItem, 0, len(items))
	for _, item := range items {
		text, read := textPerDocument[item.Metadata.Hash]
		if read {
			item = item.carryingTheText(text)
		}
		carryingItems = append(carryingItems, item)
	}

	return carryingItems
}

func (a AnsweredQuery) ItemOfEachAnsweredDocument() []AnsweredItem {
	merged := mergedItems{placeOfTakenDocument: map[yacymodel.URLHash]int{}}
	for round := range amountOfRoundsAcross(a.ItemsInTheOrderOfEachAnswer) {
		for _, itemsOfOneAnswer := range a.ItemsInTheOrderOfEachAnswer {
			if round >= len(itemsOfOneAnswer) {
				continue
			}
			merged.take(itemsOfOneAnswer[round])
		}
	}
	for _, item := range a.ItemsInNoOrder {
		merged.take(item)
	}

	return merged.items
}

type mergedItems struct {
	items                []AnsweredItem
	placeOfTakenDocument map[yacymodel.URLHash]int
}

func (m *mergedItems) take(answeredItem AnsweredItem) {
	if place, taken := m.placeOfTakenDocument[answeredItem.Metadata.Hash]; taken {
		m.items[place] = m.items[place].withTheCountsOf(answeredItem)

		return
	}
	m.placeOfTakenDocument[answeredItem.Metadata.Hash] = len(m.items)
	m.items = append(m.items, answeredItem)
}

func amountOfRoundsAcross(itemsInTheOrderOfEachAnswer [][]AnsweredItem) int {
	var rounds int
	for _, itemsOfOneAnswer := range itemsInTheOrderOfEachAnswer {
		rounds = max(rounds, len(itemsOfOneAnswer))
	}

	return rounds
}
