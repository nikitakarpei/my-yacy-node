package searchquery

import (
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	shortestCompoundRun = 2
	longestCompoundRun  = 3
)

type CompoundWord struct {
	Hash       yacymodel.Hash
	WordHashes []yacymodel.Hash
}

func compoundWordsOf(tokens []token, words []string) []CompoundWord {
	var compoundWords []CompoundWord
	for runLength := shortestCompoundRun; runLength <= longestCompoundRun; runLength++ {
		for _, run := range runsOfSpokenWordsIn(tokens, words) {
			compoundWords = append(compoundWords, compoundWordsOfLengthIn(run, runLength)...)
		}
	}

	return compoundWords
}

func runsOfSpokenWordsIn(tokens []token, words []string) [][]string {
	var runs [][]string
	var run []string
	for _, spokenToken := range tokens {
		if spokenToken.excluded || !slices.Contains(words, spokenToken.word) {
			runs = appendRun(runs, run)
			run = nil

			continue
		}
		run = append(run, spokenToken.word)
	}

	return appendRun(runs, run)
}

func appendRun(runs [][]string, run []string) [][]string {
	if len(run) < shortestCompoundRun {
		return runs
	}

	return append(runs, run)
}

func compoundWordsOfLengthIn(run []string, runLength int) []CompoundWord {
	var compoundWords []CompoundWord
	for start := 0; start+runLength <= len(run); start++ {
		compoundWords = append(compoundWords, compoundWordOf(run[start:start+runLength]))
	}

	return compoundWords
}

func compoundWordOf(spokenWords []string) CompoundWord {
	return CompoundWord{
		Hash:       yacymodel.WordHash(strings.Join(spokenWords, "")),
		WordHashes: hashesOf(spokenWords),
	}
}
