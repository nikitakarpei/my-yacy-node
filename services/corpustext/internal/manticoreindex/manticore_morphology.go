package manticoreindex

func morphologyOf(language string) string {
	return languageMorphologies[language]
}

var languageMorphologies = map[string]string{
	"en": "stem_en",
	"de": "libstemmer_de",
	"fr": "libstemmer_fr",
	"ru": "libstemmer_ru",
}
