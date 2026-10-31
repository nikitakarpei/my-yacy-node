package yacymodel_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestURLComponentsCountTheWordsOfTheURLAfterItsScheme(t *testing.T) {
	for address, want := range map[string]uint8{
		"http://example.com/article":              3,
		"https://www.mensa.org/terms-conditions/": 5,
		"https://example.com/snake_case/it's":     6,
		"http://example.com:8080/a":               4,
		"https://example.com/search?q=a&b=c":      4,
		"https://example.com/a//b":                5,
		"https://example.com/":                    2,
		"example.com":                             2,
		"http://example":                          1,
		"http://":                                 1,
		"http:///":                                0,
	} {
		if got := yacymodel.URLComponentsOf(address); got != want {
			t.Errorf("URLComponentsOf(%q) = %d, want %d", address, got, want)
		}
	}
}
