// Package spamfeatures holds the features that one page gives the spam model.
package spamfeatures

import (
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel/featurehashing"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
)

const RecipeVersion = 6

const (
	textReadRunes   = 2500
	markupReadBytes = 30000
)

type Family string

const (
	Text    Family = "text"
	Address Family = "address"
	Markup  Family = "markup"
	Links   Family = "links"
	Headers Family = "headers"
)

var Families = [...]Family{Text, Address, Markup, Links, Headers}

var (
	textNgramSizes    = featurehashing.NgramSizes{Shortest: 4, Longest: 4}
	addressNgramSizes = featurehashing.NgramSizes{Shortest: 3, Longest: 5}
	markupNgramSizes  = featurehashing.NgramSizes{Shortest: 1, Longest: 2}
)

type Words map[Family][]string

type Row struct {
	HashedEntries map[Family][]featurehashing.Entry
	MetaValues    []float64
}

func RowFrom(page htmlreading.Reading, responseHeaders http.Header) Row {
	lowercaseMarkup := lowercaseMarkupOf(page)
	links := linkReadingOf(page)
	return Row{
		HashedEntries: hashedEntriesFrom(
			familyWordsOf(page, responseHeaders, links, lowercaseMarkup),
		),
		MetaValues: metaValuesOf(page, links, lowercaseMarkup),
	}
}

func lowercaseMarkupOf(page htmlreading.Reading) string {
	return strings.ToLower(page.HTML[:min(len(page.HTML), markupReadBytes)])
}

func familyWordsOf(
	page htmlreading.Reading,
	responseHeaders http.Header,
	links linkReading,
	lowercaseMarkup string,
) Words {
	return Words{
		Text: featurehashing.CharacterNgramsFrom(
			page.Title+" "+prefixOf(page.VisibleText, textReadRunes), textNgramSizes,
		),
		Address: featurehashing.CharacterNgramsFrom(
			addressTextOf(page.Address),
			addressNgramSizes,
		),
		Markup:  featurehashing.WordNgramsFrom(markupTextOf(lowercaseMarkup), markupNgramSizes),
		Links:   links.words,
		Headers: headerWordsOf(responseHeaders, lowercaseMarkup),
	}
}

func prefixOf(text string, runes int) string {
	for position := range text {
		if runes == 0 {
			return text[:position]
		}
		runes--
	}
	return text
}

func hashedEntriesFrom(words Words) map[Family][]featurehashing.Entry {
	hashed := make(map[Family][]featurehashing.Entry, len(words))
	for family, familyWords := range words {
		hashed[family] = featurehashing.EntriesFrom(familyWords)
	}
	return hashed
}

func WordsFrom(page htmlreading.Reading, responseHeaders http.Header) Words {
	return familyWordsOf(page, responseHeaders, linkReadingOf(page), lowercaseMarkupOf(page))
}
