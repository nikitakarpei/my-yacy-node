package yacyproto_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func mustHash(t *testing.T, raw string) yacymodel.Hash {
	t.Helper()
	hash, err := yacymodel.ParseHash(raw)
	if err != nil {
		t.Fatal(err)
	}

	return hash
}

func mustLanguage(t *testing.T, raw string) yacymodel.Optional[yacymodel.Language] {
	t.Helper()
	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		t.Fatal(err)
	}

	return yacymodel.Some(language)
}

func sampleHash(tb testing.TB, word string) yacymodel.Hash {
	tb.Helper()

	hash := yacymodel.WordHash(word)
	if hash.IsZero() {
		tb.Fatalf("sample hash for %q is invalid: %q", word, hash)
	}

	return hash
}

func sampleSeed(tb testing.TB, word, name string) yacymodel.Seed {
	tb.Helper()

	peerName, err := yacymodel.ParsePeerName(name)
	if err != nil {
		tb.Fatalf("sample seed name %q is invalid: %v", name, err)
	}

	return yacymodel.Seed{
		Hash:     sampleHash(tb, word),
		Name:     peerName,
		PeerType: yacymodel.PeerSenior,
		Tags:     yacymodel.MatchAllTags(),
	}
}

func sampleURLHash(tb testing.TB, word string) yacymodel.URLHash {
	tb.Helper()

	hash, err := yacymodel.ParseURLHash(sampleHash(tb, word).String())
	if err != nil {
		tb.Fatalf("sample url hash for %q: %v", word, err)
	}

	return hash
}

func sampleRWIPosting(tb testing.TB, word, urlWord string) yacymodel.RWIPosting {
	tb.Helper()

	return yacymodel.RWIPosting{
		WordHash:   sampleHash(tb, word),
		URLHash:    sampleURLHash(tb, urlWord),
		LocalLinks: 2,
	}
}

func sampleURLMetadata(urlWord string) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Address:      "https://example.org/" + urlWord,
		Title:        urlWord,
		DocumentType: yacymodel.DocumentTypeText,
		Loaded:       yacymodel.Some(yacymodel.NewCalendarDay(2026, time.July, 18)),
		LocalLinks:   2,
	}
}

func sampleURLMetadataWireForm(tb testing.TB, metadata yacymodel.URLMetadata) string {
	tb.Helper()

	form := yacyproto.TransferURLRequest{
		URLCount: 1,
		URLs:     []yacymodel.URLMetadata{metadata},
	}.Form()

	return form.Get("url0")
}
