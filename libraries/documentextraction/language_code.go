package documentextraction

import "strings"

func LanguageCodeOf(languageTag string) string {
	code := strings.ToLower(strings.TrimSpace(languageTag))
	if dash := strings.IndexByte(code, '-'); dash >= 0 {
		code = code[:dash]
	}
	if len(code) != 2 {
		return ""
	}
	for _, letter := range code {
		if letter < 'a' || letter > 'z' {
			return ""
		}
	}
	return code
}
