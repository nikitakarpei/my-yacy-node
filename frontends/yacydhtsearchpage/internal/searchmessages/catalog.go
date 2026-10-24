// Package searchmessages holds every text the search page shows, in English, each under one
// key. A text can hold placeholders that the page fills when it shows the text.
package searchmessages

import (
	"errors"
	"fmt"
)

var errUnknownKey = errors.New("no search page text has the key")

type Catalog struct {
	texts map[string]string
}

func English() Catalog {
	return Catalog{texts: englishTexts}
}

func (catalog Catalog) TextOf(key string, values ...any) (string, error) {
	text, found := catalog.texts[key]
	if !found {
		return "", fmt.Errorf("%w %q", errUnknownKey, key)
	}
	if len(values) == 0 {
		return text, nil
	}
	return fmt.Sprintf(text, values...), nil
}
