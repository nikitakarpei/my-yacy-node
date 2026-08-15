package yacymodel_test

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestURLMetadataHashDerivesFromAddress(t *testing.T) {
	metadata := yacymodel.URLMetadata{Address: "http://example.com/a"}

	got, err := metadata.Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if want := hashOfAddress(t, "http://example.com/a"); got != want {
		t.Errorf("Hash = %q, want %q", got, want)
	}
}

func TestFreshnessPrefersLoadedThenModifiedThenFreshUntil(t *testing.T) {
	loaded := yacymodel.Some(yacymodel.NewCalendarDay(2025, time.February, 3))
	modified := yacymodel.Some(yacymodel.NewCalendarDay(2024, time.January, 2))
	freshUntil := yacymodel.Some(yacymodel.NewCalendarDay(2026, time.March, 4))

	for _, c := range []struct {
		name     string
		metadata yacymodel.URLMetadata
		want     yacymodel.Optional[yacymodel.CalendarDay]
	}{
		{
			name:     "loaded wins",
			metadata: yacymodel.URLMetadata{Loaded: loaded, Modified: modified, FreshUntil: freshUntil},
			want:     loaded,
		},
		{
			name:     "modified without loaded",
			metadata: yacymodel.URLMetadata{Modified: modified, FreshUntil: freshUntil},
			want:     modified,
		},
		{
			name:     "fresh until alone",
			metadata: yacymodel.URLMetadata{FreshUntil: freshUntil},
			want:     freshUntil,
		},
		{
			name:     "no day at all",
			metadata: yacymodel.URLMetadata{},
			want:     yacymodel.None[yacymodel.CalendarDay](),
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.metadata.Freshness(); got != c.want {
				t.Errorf("Freshness = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestIsDirectoryListingFollowsTags(t *testing.T) {
	listing := yacymodel.URLMetadata{Tags: []string{"news", "www.indexof.pages"}}
	if !listing.IsDirectoryListing() {
		t.Error("tag containing indexof should mark a directory listing")
	}
	plain := yacymodel.URLMetadata{Tags: []string{"news"}}
	if plain.IsDirectoryListing() {
		t.Error("unrelated tags should not mark a directory listing")
	}
}

func TestHasLocationFollowsCoordinates(t *testing.T) {
	located := yacymodel.URLMetadata{
		Location: yacymodel.Some(yacymodel.Coordinates{Latitude: 52.52, Longitude: 13.405}),
	}
	if !located.HasLocation() {
		t.Error("coordinates should mark a location")
	}
	if (yacymodel.URLMetadata{}).HasLocation() {
		t.Error("zero coordinates should not mark a location")
	}
}

func TestMediaPredicatesFollowLinksAndMediaType(t *testing.T) {
	for _, c := range []struct {
		name      string
		linked    yacymodel.URLMetadata
		mediaType string
		predicate func(yacymodel.URLMetadata) bool
	}{
		{
			"image",
			yacymodel.URLMetadata{ImageLinks: 1},
			"image/png",
			yacymodel.URLMetadata.HasImage,
		},
		{
			"audio",
			yacymodel.URLMetadata{AudioLinks: 1},
			"audio/mpeg",
			yacymodel.URLMetadata.HasAudio,
		},
		{
			"video",
			yacymodel.URLMetadata{VideoLinks: 1},
			"video/mp4",
			yacymodel.URLMetadata.HasVideo,
		},
		{
			"application",
			yacymodel.URLMetadata{ApplicationLinks: 1},
			"application/pdf",
			yacymodel.URLMetadata.HasApplication,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if !c.predicate(c.linked) {
				t.Error("a link of this kind should satisfy the predicate")
			}
			if !c.predicate(yacymodel.URLMetadata{MediaType: c.mediaType}) {
				t.Error("this media type should satisfy the predicate")
			}
			if c.predicate(yacymodel.URLMetadata{MediaType: "text/html"}) {
				t.Error("text/html should not satisfy the predicate")
			}
		})
	}
}
