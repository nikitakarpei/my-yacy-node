package yacyproto_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func sampleHash(tb testing.TB, word string) yacymodel.Hash {
	tb.Helper()

	hash := yacymodel.WordHash(word)
	if !hash.Valid() {
		tb.Fatalf("sample hash for %q is invalid: %q", word, hash)
	}

	return hash
}

func sampleSeed(tb testing.TB, word, name string) yacymodel.Seed {
	tb.Helper()

	seed := yacymodel.Seed{
		Hash:     sampleHash(tb, word),
		Name:     yacymodel.Some(name),
		PeerType: yacymodel.Some(yacymodel.PeerSenior),
	}

	roundTrip, err := yacymodel.ParseSeed(tb.Context(), seed.String())
	if err != nil {
		tb.Fatalf("sample seed does not round-trip: %v", err)
	}

	return roundTrip
}

func sampleRWIPosting(tb testing.TB, word, urlWord string) yacymodel.RWIPosting {
	tb.Helper()

	return yacymodel.RWIPosting{
		WordHash:   sampleHash(tb, word),
		URLHash:    yacymodel.URLHash(sampleHash(tb, urlWord)),
		LocalLinks: 2,
	}
}

func sampleURLMetadata(urlWord string) yacymodel.URLMetadata {
	return yacymodel.URLMetadata{
		Address:      "https://example.org/" + urlWord,
		Title:        urlWord,
		DocumentType: yacymodel.DocumentTypeText,
		Loaded:       yacymodel.NewCalendarDay(2026, time.July, 18),
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
