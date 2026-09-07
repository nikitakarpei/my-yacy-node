package yacymodel

const MinimumIndexedWordLength = 2

func WordIsIndexed(word string) bool {
	return len([]rune(word)) >= MinimumIndexedWordLength
}
