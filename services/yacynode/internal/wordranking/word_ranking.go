// Package wordranking reads the operator's word ranking: a file of word hashes,
// one per line, most common first. No file ranks no words.
package wordranking

import (
	"bufio"
	"fmt"
	"os"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func WordsFrom(path string) ([]yacymodel.Hash, error) {
	if path == "" {
		return nil, nil
	}
	file, err := os.Open(path) //nolint:gosec // G304: the operator names the ranking file.
	if err != nil {
		return nil, fmt.Errorf("open word ranking: %w", err)
	}
	defer func() { _ = file.Close() }()

	return wordsListedIn(bufio.NewScanner(file))
}

func wordsListedIn(lines *bufio.Scanner) ([]yacymodel.Hash, error) {
	var words []yacymodel.Hash
	for lines.Scan() {
		word, err := yacymodel.ParseHash(lines.Text())
		if err != nil {
			return nil, fmt.Errorf("read word ranking line %d: %w", len(words)+1, err)
		}
		words = append(words, word)
	}
	if err := lines.Err(); err != nil {
		return nil, fmt.Errorf("read word ranking: %w", err)
	}

	return words, nil
}
