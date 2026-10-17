// Package pagefingerprint tells versions of a page apart: a crawl writes the
// same fingerprint into its postings and its URL metadata, so equal
// fingerprints mean the same version. Two versions can share a fingerprint;
// only the modified day says which version is newer.
package pagefingerprint

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// YaCy stores a posting's link and title-word counts in one byte, so they wrap at this modulus.
const postingCountModulus = 256

type Fingerprint struct {
	Modified      yacymodel.MicroDate
	LocalLinks    int
	ExternalLinks int
	TitleWords    int
	Language      yacymodel.Language
	DocumentType  yacymodel.DocumentType
}

func FromPosting(posting yacymodel.RWIPosting) Fingerprint {
	return Fingerprint{
		Modified:      posting.LastModified,
		LocalLinks:    posting.LocalLinks % postingCountModulus,
		ExternalLinks: posting.ExternalLinks % postingCountModulus,
		TitleWords:    posting.TitleWords % postingCountModulus,
		Language:      posting.Language,
		DocumentType:  posting.DocumentType,
	}
}

func FromMetadata(metadata yacymodel.URLMetadata) Fingerprint {
	return Fingerprint{
		Modified:      modifiedDayOf(metadata),
		LocalLinks:    metadata.LocalLinks % postingCountModulus,
		ExternalLinks: metadata.ExternalLinks % postingCountModulus,
		TitleWords:    yacymodel.TitleWordsOf(metadata.Title) % postingCountModulus,
		Language:      declaredLanguageOf(metadata),
		DocumentType:  metadata.DocumentType,
	}
}

func (f Fingerprint) IsDatedAfter(now time.Time) bool {
	return f.Modified > yacymodel.MicroDateFromTime(now)
}

func modifiedDayOf(metadata yacymodel.URLMetadata) yacymodel.MicroDate {
	for _, day := range []yacymodel.Optional[yacymodel.CalendarDay]{metadata.Modified, metadata.Loaded} {
		if present, ok := day.Get(); ok {
			return yacymodel.MicroDateFromTime(present.Time())
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
