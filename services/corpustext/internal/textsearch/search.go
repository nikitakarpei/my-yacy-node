package textsearch

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

const (
	DefaultMatchLimit = 10
	MatchLimitMaximum = 50
)

type Search struct {
	Query    string
	Language string
	Offset   int
	Limit    int
}

func SearchFor(query, language string, offset, limit int) (Search, error) {
	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery == "" {
		return Search{}, errors.New("query: must hold a word")
	}
	if offset < 0 {
		return Search{}, fmt.Errorf("offset %d: must not be negative", offset)
	}
	if limit < 1 {
		return Search{}, fmt.Errorf("limit %d: must be positive", limit)
	}
	return Search{
		Query:    trimmedQuery,
		Language: documentextraction.LanguageCodeOf(language),
		Offset:   offset,
		Limit:    min(limit, MatchLimitMaximum),
	}, nil
}
