// Package consolemessages holds every text the crawl console shows, in English, each under one
// key. A text can hold placeholders that the console fills when it shows the text.
package consolemessages

import (
	"errors"
	"fmt"
)

var errUnknownKey = errors.New("no console text has the key")

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
