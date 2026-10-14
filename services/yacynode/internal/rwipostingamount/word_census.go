package rwipostingamount

import (
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const wordsCountedInFull = 10_000

var sampledWordPrefixes = sampledWordPrefixesOf(yacymodel.Alphabet)

type wordCensus struct {
	amountPerWord *vault.CountIndex[yacymodel.RWIPosting, yacymodel.Hash]
}

func (c wordCensus) EstimatedAmountOfWords(tx *vault.Txn) (int, error) {
	countedWords, isComplete, err := c.wordsCountedUpTo(tx, wordsCountedInFull)
	if err != nil || isComplete {
		return countedWords, err
	}
	sampledWords, err := c.wordsInSample(tx)

	return sampledWords * len(yacymodel.Alphabet), err
}

func (c wordCensus) wordsCountedUpTo(
	tx *vault.Txn,
	limit int,
) (countedWords int, isComplete bool, err error) {
	err = c.amountPerWord.Scan(tx, vault.EveryKey(), func(yacymodel.Hash, int) (bool, error) {
		countedWords++

		return countedWords <= limit, nil
	})
	if err != nil {
		return 0, false, fmt.Errorf("count words: %w", err)
	}

	return countedWords, countedWords <= limit, nil
}

func (c wordCensus) wordsInSample(tx *vault.Txn) (int, error) {
	var sampledWords int
	for _, prefix := range sampledWordPrefixes {
		err := c.amountPerWord.Scan(
			tx,
			wordKeys.KeysFromFirst(lowestWordWith(prefix)),
			func(word yacymodel.Hash, _ int) (bool, error) {
				if !strings.HasPrefix(word.String(), prefix) {
					return false, nil
				}
				sampledWords++

				return true, nil
			},
		)
		if err != nil {
			return 0, fmt.Errorf("count sampled words: %w", err)
		}
	}

	return sampledWords, nil
}

func lowestWordWith(prefix string) yacymodel.Hash {
	lowestSymbol := yacymodel.Alphabet[:1]
	word, _ := yacymodel.ParseHash(
		prefix + strings.Repeat(lowestSymbol, yacymodel.HashLength-len(prefix)),
	)

	return word
}

func sampledWordPrefixesOf(alphabet string) []string {
	sampledSymbol := alphabet[:1]
	prefixes := make([]string, 0, len(alphabet)*len(alphabet))
	for _, first := range alphabet {
		for _, second := range alphabet {
			prefixes = append(prefixes, string(first)+string(second)+sampledSymbol)
		}
	}

	return prefixes
}
