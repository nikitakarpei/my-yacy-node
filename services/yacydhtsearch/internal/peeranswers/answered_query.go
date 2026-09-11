// Package peeranswers holds what the peers answered for one whole query: the
// items of each answer in the order the peer that answered put them, the items
// that came back in no order, how many documents the peers that counted each
// query word hold for it, and the one item of each answered document across
// every answer. An item carries the metadata a peer holds for the document and
// how often a peer counted each query word the document matched, beside how
// many words its text holds. A peer asked about one word counts that word; a
// document that matched a word no peer counted still matched the word, and
// carries the zero count for it. The text of a document, when that document was
// read, replaces those counts and the snippet on every item of the document.
package peeranswers

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type AnsweredQuery struct {
	ItemsInTheOrderOfEachAnswer [][]AnsweredItem
	ItemsInNoOrder              []AnsweredItem
	DocumentsHeldPerQueryWord   map[yacymodel.Hash]int
}

func (a AnsweredQuery) CarryingTheTextOfEachDocument(
	textPerDocument map[yacymodel.URLHash]DocumentText,
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
	textPerDocument map[yacymodel.URLHash]DocumentText,
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
