package urlmeta

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type urlMetadataValueCodec struct{}

func (urlMetadataValueCodec) Encode(metadata yacymodel.URLMetadata) ([]byte, error) {
	var stored storedfields.Writer
	stored.Text(metadata.Address)
	writeReferrer(&stored, metadata.Referrer)
	stored.Text(metadata.Title)
	stored.Text(metadata.Author)
	stored.Count(len(metadata.Tags))
	for _, tag := range metadata.Tags {
		stored.Text(tag)
	}
	stored.Text(metadata.Publisher)
	writeLocation(&stored, metadata.Location)
	writeDay(&stored, metadata.Modified)
	writeDay(&stored, metadata.Loaded)
	writeDay(&stored, metadata.FreshUntil)
	stored.Byte(byte(metadata.DocumentType))
	stored.Text(metadata.MediaType)
	writeLanguage(&stored, metadata.Language)
	stored.Count(metadata.ByteSize)
	stored.Count(metadata.WordCount)
	stored.Count(metadata.LocalLinks)
	stored.Count(metadata.ExternalLinks)
	stored.Count(metadata.ImageLinks)
	stored.Count(metadata.AudioLinks)
	stored.Count(metadata.VideoLinks)
	stored.Count(metadata.ApplicationLinks)
	stored.Text(metadata.Snippet)
	stored.Text(metadata.FaviconAddress)

	return stored.Record(), nil
}

func (urlMetadataValueCodec) Decode(data []byte) (yacymodel.URLMetadata, error) {
	if len(data) == 0 {
		return yacymodel.URLMetadata{}, fmt.Errorf(
			"%w: empty url metadata value",
			yacymodel.ErrBadURLMetadata,
		)
	}
	stored := storedfields.ReaderOf(data, yacymodel.ErrBadURLMetadata)
	metadata := yacymodel.URLMetadata{
		Address:          stored.Text("address"),
		Referrer:         referrerFrom(stored),
		Title:            stored.Text("title"),
		Author:           stored.Text("author"),
		Tags:             tagsFrom(stored),
		Publisher:        stored.Text("publisher"),
		Location:         locationFrom(stored),
		Modified:         dayFrom(stored, "modified"),
		Loaded:           dayFrom(stored, "loaded"),
		FreshUntil:       dayFrom(stored, "fresh until"),
		DocumentType:     yacymodel.DocumentType(stored.Byte("content type")),
		MediaType:        stored.Text("media type"),
		Language:         languageFrom(stored),
		ByteSize:         stored.Count("byte size"),
		WordCount:        stored.Count("word count"),
		LocalLinks:       stored.Count("local links"),
		ExternalLinks:    stored.Count("external links"),
		ImageLinks:       stored.Count("image links"),
		AudioLinks:       stored.Count("audio links"),
		VideoLinks:       stored.Count("video links"),
		ApplicationLinks: stored.Count("application links"),
		Snippet:          stored.Text("snippet"),
		FaviconAddress:   stored.Text("favicon address"),
	}
	if err := stored.Err(); err != nil {
		return yacymodel.URLMetadata{}, err
	}

	return metadata, nil
}

func writeReferrer(stored *storedfields.Writer, referrer yacymodel.Optional[yacymodel.URLHash]) {
	hash, known := referrer.Get()
	stored.Presence(known)
	if !known {
		return
	}
	stored.Fixed(hash.Bytes())
}

func referrerFrom(stored *storedfields.Reader) yacymodel.Optional[yacymodel.URLHash] {
	if !stored.Presence("referrer") {
		return yacymodel.None[yacymodel.URLHash]()
	}
	raw := stored.Fixed("referrer", yacymodel.HashByteLength)
	referrer, err := yacymodel.ParseURLHashBytes(raw)
	if err != nil {
		stored.Reject("referrer", err)

		return yacymodel.None[yacymodel.URLHash]()
	}

	return yacymodel.Some(referrer)
}

func writeLanguage(stored *storedfields.Writer, language yacymodel.Optional[yacymodel.Language]) {
	code, spoken := language.Get()
	if !spoken {
		stored.Text("")

		return
	}
	stored.Text(code.String())
}

func languageFrom(stored *storedfields.Reader) yacymodel.Optional[yacymodel.Language] {
	raw := stored.Text("language")
	if raw == "" {
		return yacymodel.None[yacymodel.Language]()
	}
	language, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		stored.Reject("language", err)

		return yacymodel.None[yacymodel.Language]()
	}

	return yacymodel.Some(language)
}

func writeLocation(
	stored *storedfields.Writer,
	location yacymodel.Optional[yacymodel.Coordinates],
) {
	coordinates, placed := location.Get()
	stored.Presence(placed)
	if !placed {
		return
	}
	stored.Float(coordinates.Latitude)
	stored.Float(coordinates.Longitude)
}

func locationFrom(stored *storedfields.Reader) yacymodel.Optional[yacymodel.Coordinates] {
	if !stored.Presence("location") {
		return yacymodel.None[yacymodel.Coordinates]()
	}
	coordinates, err := yacymodel.NewCoordinates(
		stored.Float("latitude"),
		stored.Float("longitude"),
	)
	if err != nil {
		stored.Reject("location", err)

		return yacymodel.None[yacymodel.Coordinates]()
	}

	return yacymodel.Some(coordinates)
}

func writeDay(stored *storedfields.Writer, day yacymodel.Optional[yacymodel.CalendarDay]) {
	calendarDay, dated := day.Get()
	stored.Presence(dated)
	if !dated {
		return
	}
	stored.Count(calendarDay.Year)
	stored.Count(int(calendarDay.Month))
	stored.Count(calendarDay.Day)
}

func dayFrom(stored *storedfields.Reader, field string) yacymodel.Optional[yacymodel.CalendarDay] {
	if !stored.Presence(field) {
		return yacymodel.None[yacymodel.CalendarDay]()
	}
	year := stored.Count(field + " year")
	month := stored.Count(field + " month")
	dayOfMonth := stored.Count(field + " day")

	return yacymodel.Some(yacymodel.NewCalendarDay(year, time.Month(month), dayOfMonth))
}

func tagsFrom(stored *storedfields.Reader) []string {
	count := stored.Count("tag count")
	if count <= 0 {
		return nil
	}
	if count > stored.BytesLeft() {
		stored.Reject("tag count", fmt.Errorf("%d tags exceed the bytes left", count))

		return nil
	}
	tags := make([]string, 0, count)
	for range count {
		tags = append(tags, stored.Text("tag"))
	}
	if stored.Err() != nil {
		return nil
	}

	return tags
}
