package yacymodel

func TitleWordsOf(title string) int {
	if title == "" {
		return 1
	}
	words := 0
	inWord := false
	for i := range len(title) {
		if title[i] == ' ' {
			inWord = false

			continue
		}
		if !inWord {
			words++
			inWord = true
		}
	}
	if title[0] == ' ' && words > 0 {
		words++
	}

	return words
}
