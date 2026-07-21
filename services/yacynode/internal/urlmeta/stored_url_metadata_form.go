package urlmeta

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const storedURLMetadataFormat byte = 0x02

// storedURLMetadataCodec translates between the url metadata domain type and
// this node's on-disk row.
type storedURLMetadataCodec struct{}

func (storedURLMetadataCodec) Encode(metadata yacymodel.URLMetadata) ([]byte, error) {
	var w urlMetadataWriter
	w.uint8(storedURLMetadataFormat)
	w.text(metadata.Address)
	w.referrer(metadata.Referrer)
	w.text(metadata.Title)
	w.text(metadata.Author)
	w.count(len(metadata.Tags))
	for _, tag := range metadata.Tags {
		w.text(tag)
	}
	w.text(metadata.Publisher)
	w.location(metadata.Location)
	w.day(metadata.Modified)
	w.day(metadata.Loaded)
	w.day(metadata.FreshUntil)
	w.uint8(byte(metadata.DocumentType))
	w.text(metadata.MediaType)
	w.language(metadata.Language)
	w.count(metadata.ByteSize)
	w.count(metadata.WordCount)
	w.count(metadata.LocalLinks)
	w.count(metadata.ExternalLinks)
	w.count(metadata.ImageLinks)
	w.count(metadata.AudioLinks)
	w.count(metadata.VideoLinks)
	w.count(metadata.ApplicationLinks)
	w.text(metadata.Snippet)
	w.text(metadata.FaviconAddress)

	return w.bytes(), nil
}

func (storedURLMetadataCodec) Decode(data []byte) (yacymodel.URLMetadata, error) {
	if len(data) == 0 {
		return yacymodel.URLMetadata{}, fmt.Errorf(
			"%w: empty url metadata value",
			yacymodel.ErrBadURLMetadata,
		)
	}
	if data[0] != storedURLMetadataFormat {
		return yacymodel.URLMetadata{}, fmt.Errorf(
			"%w: unknown stored url metadata format 0x%02x, want 0x%02x",
			yacymodel.ErrBadURLMetadata,
			data[0],
			storedURLMetadataFormat,
		)
	}

	r := newURLMetadataReader(data[1:])
	metadata := yacymodel.URLMetadata{Address: r.text("address")}
	referrer, referrerErr := r.referrer()
	metadata.Referrer = referrer
	metadata.Title = r.text("title")
	metadata.Author = r.text("author")
	metadata.Tags = r.tags()
	metadata.Publisher = r.text("publisher")
	location, locationErr := r.location()
	metadata.Location = location
	metadata.Modified = r.day("modified")
	metadata.Loaded = r.day("loaded")
	metadata.FreshUntil = r.day("fresh until")
	metadata.DocumentType = yacymodel.DocumentType(r.uint8("content type"))
	metadata.MediaType = r.text("media type")
	language, languageErr := r.language()
	metadata.Language = language
	metadata.ByteSize = r.count("byte size")
	metadata.WordCount = r.count("word count")
	metadata.LocalLinks = r.count("local links")
	metadata.ExternalLinks = r.count("external links")
	metadata.ImageLinks = r.count("image links")
	metadata.AudioLinks = r.count("audio links")
	metadata.VideoLinks = r.count("video links")
	metadata.ApplicationLinks = r.count("application links")
	metadata.Snippet = r.text("snippet")
	metadata.FaviconAddress = r.text("favicon address")

	if r.err != nil {
		return yacymodel.URLMetadata{}, r.err
	}
	for _, err := range []error{referrerErr, locationErr, languageErr} {
		if err != nil {
			return yacymodel.URLMetadata{}, fmt.Errorf(
				"%w: %w", yacymodel.ErrBadURLMetadata, err,
			)
		}
	}

	return metadata, nil
}

func (w *urlMetadataWriter) referrer(referrer yacymodel.Optional[yacymodel.URLHash]) {
	if value, ok := referrer.Get(); ok {
		w.text(value.String())

		return
	}
	w.text("")
}

func (r *urlMetadataReader) referrer() (yacymodel.Optional[yacymodel.URLHash], error) {
	raw := r.text("referrer")
	if raw == "" {
		return yacymodel.None[yacymodel.URLHash](), nil
	}
	referrer, err := yacymodel.ParseURLHash(raw)
	if err != nil {
		return yacymodel.None[yacymodel.URLHash](), err
	}

	return yacymodel.Some(referrer), nil
}

func (w *urlMetadataWriter) language(language yacymodel.Optional[yacymodel.Language]) {
	if code, ok := language.Get(); ok {
		w.text(code.String())

		return
	}
	w.text("")
}

func (r *urlMetadataReader) language() (yacymodel.Optional[yacymodel.Language], error) {
	raw := r.text("language")
	if raw == "" {
		return yacymodel.None[yacymodel.Language](), nil
	}
	code, err := yacymodel.ParseLanguage(raw)
	if err != nil {
		return yacymodel.None[yacymodel.Language](), err
	}

	return yacymodel.Some(code), nil
}

func (w *urlMetadataWriter) location(location yacymodel.Optional[yacymodel.Coordinates]) {
	coordinates, _ := location.Get()
	w.degrees(coordinates.Latitude)
	w.degrees(coordinates.Longitude)
}

func (r *urlMetadataReader) location() (yacymodel.Optional[yacymodel.Coordinates], error) {
	latitude := r.degrees("latitude")
	longitude := r.degrees("longitude")
	if latitude == 0 && longitude == 0 {
		return yacymodel.None[yacymodel.Coordinates](), nil
	}
	coordinates, err := yacymodel.NewCoordinates(latitude, longitude)
	if err != nil {
		return yacymodel.None[yacymodel.Coordinates](), err
	}

	return yacymodel.Some(coordinates), nil
}

func (w *urlMetadataWriter) day(day yacymodel.Optional[yacymodel.CalendarDay]) {
	value, _ := day.Get()
	w.count(value.Year)
	w.count(int(value.Month))
	w.count(value.Day)
}

func (r *urlMetadataReader) day(field string) yacymodel.Optional[yacymodel.CalendarDay] {
	year := r.count(field + " year")
	month := r.count(field + " month")
	dayOfMonth := r.count(field + " day")
	if year == 0 {
		return yacymodel.None[yacymodel.CalendarDay]()
	}

	return yacymodel.Some(yacymodel.NewCalendarDay(year, time.Month(month), dayOfMonth))
}

func (r *urlMetadataReader) tags() []string {
	count := r.count("tag count")
	if r.err != nil || count <= 0 {
		return nil
	}
	tags := make([]string, 0, count)
	for range count {
		tags = append(tags, r.text("tag"))
		if r.err != nil {
			return nil
		}
	}

	return tags
}
