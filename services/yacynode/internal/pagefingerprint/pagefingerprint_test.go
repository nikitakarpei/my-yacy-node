package pagefingerprint_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
)

var crawlDay = yacymodel.NewCalendarDay(2020, time.January, 1)

func germanLanguage(t *testing.T) yacymodel.Language {
	t.Helper()

	language, err := yacymodel.ParseLanguage("de")
	if err != nil {
		t.Fatalf("ParseLanguage: %v", err)
	}

	return language
}

func TestPostingAndMetadataOfOneCrawlShareAFingerprint(t *testing.T) {
	german := germanLanguage(t)
	metadata := yacymodel.URLMetadata{
		Title:         "Hello World",
		Modified:      yacymodel.Some(crawlDay),
		Loaded:        yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 16)),
		Language:      yacymodel.Some(german),
		DocumentType:  yacymodel.DocumentTypeText,
		LocalLinks:    300,
		ExternalLinks: 4,
	}
	posting := yacymodel.RWIPosting{
		LastModified:  yacymodel.MicroDateFromTime(crawlDay.Time()),
		TitleWords:    2,
		Language:      german,
		DocumentType:  yacymodel.DocumentTypeText,
		LocalLinks:    300 % 256,
		ExternalLinks: 4,
	}

	if pagefingerprint.OfPosting(posting) != pagefingerprint.OfMetadata(metadata) {
		t.Fatalf("fingerprints differ:\n posting  %+v\n metadata %+v",
			pagefingerprint.OfPosting(posting), pagefingerprint.OfMetadata(metadata))
	}

	posting.ExternalLinks = 5
	if pagefingerprint.OfPosting(posting) == pagefingerprint.OfMetadata(metadata) {
		t.Fatal("a posting with other link counts shares the fingerprint")
	}
}

func TestMetadataModifiedDayFallsBackToTheLoadedDay(t *testing.T) {
	loaded := yacymodel.NewCalendarDay(2025, time.February, 3)

	for name, c := range map[string]struct {
		metadata yacymodel.URLMetadata
		want     yacymodel.MicroDate
	}{
		"modified wins": {
			metadata: yacymodel.URLMetadata{Modified: yacymodel.Some(crawlDay), Loaded: yacymodel.Some(loaded)},
			want:     yacymodel.MicroDateFromTime(crawlDay.Time()),
		},
		"loaded without modified": {
			metadata: yacymodel.URLMetadata{Loaded: yacymodel.Some(loaded)},
			want:     yacymodel.MicroDateFromTime(loaded.Time()),
		},
		"neither day": {metadata: yacymodel.URLMetadata{}, want: 0},
	} {
		if got := pagefingerprint.OfMetadata(c.metadata).Modified; got != c.want {
			t.Errorf("%s: modified = %d, want %d", name, got, c.want)
		}
	}
}

func TestMetadataWithoutLanguageMatchesAnUndeclaredDocument(t *testing.T) {
	fingerprint := pagefingerprint.OfMetadata(yacymodel.URLMetadata{})

	if fingerprint.Language != yacymodel.LanguageOfUndeclaredDocument {
		t.Fatalf("language = %v, want the language of an undeclared document", fingerprint.Language)
	}
}

func TestFingerprintIsDatedAfterTheDayOfNow(t *testing.T) {
	lateOnCrawlDay := crawlDay.Time().Add(23 * time.Hour)

	for name, c := range map[string]struct {
		modified yacymodel.CalendarDay
		want     bool
	}{
		"same day":   {modified: crawlDay, want: false},
		"day before": {modified: yacymodel.NewCalendarDay(2019, time.December, 31), want: false},
		"next day":   {modified: yacymodel.NewCalendarDay(2020, time.January, 2), want: true},
	} {
		fingerprint := pagefingerprint.OfMetadata(
			yacymodel.URLMetadata{Modified: yacymodel.Some(c.modified)},
		)
		if got := fingerprint.IsDatedAfter(lateOnCrawlDay); got != c.want {
			t.Errorf("%s: dated after = %t, want %t", name, got, c.want)
		}
	}
}
