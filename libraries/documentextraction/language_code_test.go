package documentextraction_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

func TestLanguageCodeOfKeepsTheTwoLetterPrimarySubtag(t *testing.T) {
	cases := map[string]string{
		"en":    "en",
		"DE-at": "de",
		"fr-CA": "fr",
		"":      "",
		"all":   "",
		"auto":  "",
		"e1":    "",
		"en_US": "",
	}
	for languageTag, want := range cases {
		if got := documentextraction.LanguageCodeOf(languageTag); got != want {
			t.Errorf("LanguageCodeOf(%q) = %q, want %q", languageTag, got, want)
		}
	}
}
