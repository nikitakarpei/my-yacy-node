package yacymodel_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestPageRWIRootIsAbsentWithoutAPostingsRoot(t *testing.T) {
	metadata := metadataWithEveryField(t)
	metadata.PostingsRoot = yacymodel.None[yacymodel.PostingsRoot]()

	if yacymodel.PageRWIRootFrom(metadata).Present() {
		t.Error("a row without a postings root has a page RWI root")
	}
}

func TestPageRWIRootOfAKnownRow(t *testing.T) {
	root := pageRWIRootOf(t, metadataWithEveryField(t))

	const want = "cc19a541dd990c3f197f173f58c1db1cf86ad068397d3222bce6bddae9ba40c8"
	if got := hex.EncodeToString(root.Bytes()); got != want {
		t.Errorf("page RWI root = %s, want %s", got, want)
	}
}

func TestPageRWIRootDoesNotCoverTheLoadedDay(t *testing.T) {
	metadata := metadataWithEveryField(t)
	reloaded := metadataWithEveryField(t)
	reloaded.Loaded = yacymodel.Some(yacymodel.NewCalendarDay(2027, time.January, 2))

	if pageRWIRootOf(t, metadata) != pageRWIRootOf(t, reloaded) {
		t.Error("a later Loaded day changed the page RWI root")
	}
}

func TestPageRWIRootCoversEveryOtherField(t *testing.T) {
	original := pageRWIRootOf(t, metadataWithEveryField(t))
	for name, change := range fieldChanges(t) {
		changed := metadataWithEveryField(t)
		change(&changed)
		if pageRWIRootOf(t, changed) == original {
			t.Errorf("%s: the page RWI root did not change", name)
		}
	}
}

func TestPageRWIRootTellsAbsentFieldsFromEmptyOnes(t *testing.T) {
	withoutSize := metadataWithEveryField(t)
	withoutSize.ByteSize = yacymodel.None[uint32]()
	withZeroSize := metadataWithEveryField(t)
	withZeroSize.ByteSize = yacymodel.Some(uint32(0))

	if pageRWIRootOf(t, withoutSize) == pageRWIRootOf(t, withZeroSize) {
		t.Error("an absent byte size and a byte size of 0 give one page RWI root")
	}
}

func TestPageRWIRootTellsWhereTextEnds(t *testing.T) {
	oneTag := metadataWithEveryField(t)
	oneTag.Tags = []string{"ab"}
	twoTags := metadataWithEveryField(t)
	twoTags.Tags = []string{"a", "b"}
	titleText := metadataWithEveryField(t)
	titleText.Title, titleText.Author = "ab", ""
	authorText := metadataWithEveryField(t)
	authorText.Title, authorText.Author = "a", "b"

	if pageRWIRootOf(t, oneTag) == pageRWIRootOf(t, twoTags) {
		t.Error("tags [ab] and [a b] give one page RWI root")
	}
	if pageRWIRootOf(t, titleText) == pageRWIRootOf(t, authorText) {
		t.Error("text moved from title to author gives one page RWI root")
	}
}

func TestPageRWIRootTestsNameEveryField(t *testing.T) {
	fieldsOutsideTheRoot := []string{"Loaded"}
	changedFields := fieldChanges(t)
	for field := range reflect.TypeFor[yacymodel.URLMetadata]().Fields() {
		_, changed := changedFields[field.Name]
		if !changed && !slices.Contains(fieldsOutsideTheRoot, field.Name) {
			t.Errorf(
				"field %s is neither in the page RWI root tests nor left out of the root",
				field.Name,
			)
		}
	}
}

func fieldChanges(t *testing.T) map[string]func(*yacymodel.URLMetadata) {
	t.Helper()
	otherHash := urlHashOfByte(t, 'B')
	otherLanguage := languageOf(t, "fr")
	otherLocation, err := yacymodel.NewCoordinates(1, 2)
	if err != nil {
		t.Fatal(err)
	}
	otherDay := yacymodel.Some(yacymodel.NewCalendarDay(2020, time.March, 4))

	return map[string]func(*yacymodel.URLMetadata){
		"Hash":             func(m *yacymodel.URLMetadata) { m.Hash = otherHash },
		"Referrer":         func(m *yacymodel.URLMetadata) { m.Referrer = yacymodel.Some(otherHash) },
		"DocumentType":     func(m *yacymodel.URLMetadata) { m.DocumentType = yacymodel.DocumentTypeImage },
		"Language":         func(m *yacymodel.URLMetadata) { m.Language = yacymodel.Some(otherLanguage) },
		"PostingsRoot":     func(m *yacymodel.URLMetadata) { m.PostingsRoot = otherPostingsRoot(t) },
		"ByteSize":         func(m *yacymodel.URLMetadata) { m.ByteSize = yacymodel.Some(uint32(1)) },
		"WordCount":        func(m *yacymodel.URLMetadata) { m.WordCount++ },
		"LocalLinks":       func(m *yacymodel.URLMetadata) { m.LocalLinks++ },
		"ExternalLinks":    func(m *yacymodel.URLMetadata) { m.ExternalLinks++ },
		"ImageLinks":       func(m *yacymodel.URLMetadata) { m.ImageLinks++ },
		"AudioLinks":       func(m *yacymodel.URLMetadata) { m.AudioLinks++ },
		"VideoLinks":       func(m *yacymodel.URLMetadata) { m.VideoLinks++ },
		"ApplicationLinks": func(m *yacymodel.URLMetadata) { m.ApplicationLinks++ },
		"Address":          func(m *yacymodel.URLMetadata) { m.Address += "x" },
		"Title":            func(m *yacymodel.URLMetadata) { m.Title += "x" },
		"Author":           func(m *yacymodel.URLMetadata) { m.Author += "x" },
		"Tags":             func(m *yacymodel.URLMetadata) { m.Tags = append(m.Tags, "x") },
		"Publisher":        func(m *yacymodel.URLMetadata) { m.Publisher += "x" },
		"Location":         func(m *yacymodel.URLMetadata) { m.Location = yacymodel.Some(otherLocation) },
		"Modified":         func(m *yacymodel.URLMetadata) { m.Modified = otherDay },
		"FreshUntil":       func(m *yacymodel.URLMetadata) { m.FreshUntil = otherDay },
		"MediaType":        func(m *yacymodel.URLMetadata) { m.MediaType += "x" },
		"Snippet":          func(m *yacymodel.URLMetadata) { m.Snippet += "x" },
		"FaviconAddress":   func(m *yacymodel.URLMetadata) { m.FaviconAddress += "x" },
	}
}

func metadataWithEveryField(t *testing.T) yacymodel.URLMetadata {
	t.Helper()
	hash := urlHashOfByte(t, 'A')
	referrer := urlHashOfByte(t, 'C')
	location, err := yacymodel.NewCoordinates(52.5, 13.4)
	if err != nil {
		t.Fatal(err)
	}

	return yacymodel.URLMetadata{
		Hash:             hash,
		Referrer:         yacymodel.Some(referrer),
		DocumentType:     yacymodel.DocumentTypeText,
		Language:         yacymodel.Some(languageOf(t, "de")),
		PostingsRoot:     yacymodel.Some(postingsRootOfSeed(t, 1)),
		ByteSize:         yacymodel.Some(uint32(2048)),
		WordCount:        2,
		LocalLinks:       3,
		ExternalLinks:    4,
		ImageLinks:       5,
		AudioLinks:       6,
		VideoLinks:       7,
		ApplicationLinks: 8,
		Address:          "https://example.org/page",
		Title:            "Alpha and beta",
		Author:           "Ada",
		Tags:             []string{"one", "two"},
		Publisher:        "Example",
		Location:         yacymodel.Some(location),
		Modified:         yacymodel.Some(yacymodel.NewCalendarDay(2026, time.October, 30)),
		Loaded:           yacymodel.Some(yacymodel.NewCalendarDay(2026, time.November, 6)),
		FreshUntil:       yacymodel.Some(yacymodel.NewCalendarDay(2026, time.December, 1)),
		MediaType:        "text/html",
		Snippet:          "Alpha",
		FaviconAddress:   "https://example.org/favicon.ico",
	}
}

func otherPostingsRoot(t *testing.T) yacymodel.Optional[yacymodel.PostingsRoot] {
	t.Helper()

	return yacymodel.Some(postingsRootOfSeed(t, 2))
}

func postingsRootOfSeed(t *testing.T, seed uint64) yacymodel.PostingsRoot {
	t.Helper()
	postings := randomPagePostings(seededRandom(seed), 3)

	return rootOf(t, wholePagePostingsOf(t, postings, partitionsFrom(t, 4)))
}

func pageRWIRootOf(t *testing.T, metadata yacymodel.URLMetadata) yacymodel.PageRWIRoot {
	t.Helper()
	root, present := yacymodel.PageRWIRootFrom(metadata).Get()
	if !present {
		t.Fatal("no page RWI root for a row with a postings root")
	}

	return root
}

func urlHashOfByte(t *testing.T, filler byte) yacymodel.URLHash {
	t.Helper()
	hash, err := yacymodel.ParseURLHashBytes(bytes.Repeat([]byte{filler}, yacymodel.HashByteLength))
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func languageOf(t *testing.T, code string) yacymodel.Language {
	t.Helper()
	language, err := yacymodel.ParseLanguage(code)
	if err != nil {
		t.Fatal(err)
	}

	return language
}

func TestPageRWIRootBytesRoundTrip(t *testing.T) {
	root := pageRWIRootOf(t, metadataWithEveryField(t))

	parsed, err := yacymodel.ParsePageRWIRootBytes(root.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if parsed != root {
		t.Errorf("parsed root = %x, want %x", parsed.Bytes(), root.Bytes())
	}
}

func TestParsePageRWIRootBytesRefusesAnotherLength(t *testing.T) {
	if _, err := yacymodel.ParsePageRWIRootBytes(make([]byte, 31)); err == nil {
		t.Error("31 bytes parse as a page RWI root")
	}
}
