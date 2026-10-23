// Package languageindex names the per-language search indexes corpustext
// writes crawled pages into and routes a page to the index of its language.
package languageindex

import (
	"fmt"
	"strings"
)

const (
	SchemaVersion        = 1
	UndeterminedLanguage = "und"
)

type LanguageIndexes struct {
	prefix  string
	indexes []LanguageIndex
}

type LanguageIndex struct {
	Language string
	Name     string
}

func IndexesFor(baseName string, allowedLanguages []string) (LanguageIndexes, error) {
	if err := validateBaseName(baseName); err != nil {
		return LanguageIndexes{}, err
	}
	prefix := fmt.Sprintf("%s_v%d", baseName, SchemaVersion)
	indexes := []LanguageIndex{indexIn(prefix, UndeterminedLanguage)}
	for _, allowed := range allowedLanguages {
		language := primarySubtagOf(allowed)
		if !isSupported(language) {
			return LanguageIndexes{}, fmt.Errorf("language %q: not supported", allowed)
		}
		if containsLanguage(indexes, language) {
			return LanguageIndexes{}, fmt.Errorf("language %q: listed more than once", allowed)
		}
		indexes = append(indexes, indexIn(prefix, language))
	}
	return LanguageIndexes{prefix: prefix, indexes: indexes}, nil
}

func validateBaseName(baseName string) error {
	if baseName == "" {
		return fmt.Errorf("index base name: must be set")
	}
	for position, letter := range baseName {
		switch {
		case letter >= 'a' && letter <= 'z':
		case letter >= '0' && letter <= '9' && position > 0:
		case letter == '_' && position > 0:
		default:
			return fmt.Errorf(
				"index base name %q: allows lowercase letters, digits and underscores"+
					" and starts with a letter",
				baseName,
			)
		}
	}
	return nil
}

func indexIn(prefix, language string) LanguageIndex {
	return LanguageIndex{Language: language, Name: prefix + "_" + language}
}

func primarySubtagOf(language string) string {
	primary := strings.ToLower(strings.TrimSpace(language))
	if separator := strings.IndexAny(primary, "-_"); separator >= 0 {
		primary = primary[:separator]
	}
	return primary
}

func isSupported(language string) bool {
	_, supported := supportedLanguages[language]
	return supported
}

var supportedLanguages = map[string]struct{}{
	"en": {},
	"de": {},
	"fr": {},
	"ru": {},
}

func containsLanguage(indexes []LanguageIndex, language string) bool {
	for _, index := range indexes {
		if index.Language == language {
			return true
		}
	}
	return false
}

func (indexes LanguageIndexes) Prefix() string {
	return indexes.prefix
}

func (indexes LanguageIndexes) All() []LanguageIndex {
	return indexes.indexes
}

func (indexes LanguageIndexes) NameFor(documentLanguage string) string {
	language := primarySubtagOf(documentLanguage)
	for _, index := range indexes.indexes {
		if index.Language == language {
			return index.Name
		}
	}
	return indexes.indexes[0].Name
}
