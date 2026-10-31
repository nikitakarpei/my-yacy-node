// Package pagefingerprint holds the page fields that a crawl writes into both
// its postings and its URL metadata, and writes them as stored fields.
package pagefingerprint

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Fingerprint struct {
	Modified      yacymodel.MicroDate
	LocalLinks    uint8
	ExternalLinks uint8
	TitleWords    uint8
	Language      yacymodel.Language
	DocumentType  yacymodel.DocumentType
}

func FromPosting(posting yacymodel.RWIPosting) Fingerprint {
	return Fingerprint{
		Modified:      posting.LastModified,
		LocalLinks:    posting.LocalLinks,
		ExternalLinks: posting.ExternalLinks,
		TitleWords:    posting.TitleWords,
		Language:      posting.Language,
		DocumentType:  posting.DocumentType,
	}
}

func FromMetadata(metadata yacymodel.URLMetadata) Fingerprint {
	return Fingerprint{
		Modified:      modifiedDayOf(metadata),
		LocalLinks:    yacymodel.WrappedCardinalFrom(metadata.LocalLinks),
		ExternalLinks: yacymodel.WrappedCardinalFrom(metadata.ExternalLinks),
		TitleWords:    yacymodel.TitleWordsOf(metadata.Title),
		Language:      declaredLanguageOf(metadata),
		DocumentType:  metadata.DocumentType,
	}
}

func modifiedDayOf(metadata yacymodel.URLMetadata) yacymodel.MicroDate {
	for _, day := range []yacymodel.Optional[yacymodel.CalendarDay]{metadata.Modified, metadata.Loaded} {
		if present, ok := day.Get(); ok {
			if date, inRange := yacymodel.MicroDateOf(present.Time()).Get(); inRange {
				return date
			}
		}
	}

	return 0
}

func declaredLanguageOf(metadata yacymodel.URLMetadata) yacymodel.Language {
	language, declared := metadata.Language.Get()
	if !declared {
		return yacymodel.LanguageOfUndeclaredDocument
	}

	return language
}
