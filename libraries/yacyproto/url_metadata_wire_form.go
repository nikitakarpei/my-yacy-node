package yacyproto

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	urlMetadataColHash          = "hash"
	urlMetadataColAddress       = "url"
	urlMetadataColTitle         = "descr"
	urlMetadataColAuthor        = "author"
	urlMetadataColTags          = "tags"
	urlMetadataColPublisher     = "publisher"
	urlMetadataColLatitude      = "lat"
	urlMetadataColLongitude     = "lon"
	urlMetadataColModified      = "mod"
	urlMetadataColLoaded        = "load"
	urlMetadataColFreshUntil    = "fresh"
	urlMetadataColReferrer      = "referrer"
	urlMetadataColByteSize      = "size"
	urlMetadataColWordCount     = "wc"
	urlMetadataColDocType       = "dt"
	urlMetadataColMediaType     = "mime"
	urlMetadataColFlags         = "flags"
	urlMetadataColLanguage      = "lang"
	urlMetadataColLocalLinks    = "llocal"
	urlMetadataColExternalLinks = "lother"
	urlMetadataColImageLinks    = "limage"
	urlMetadataColAudioLinks    = "laudio"
	urlMetadataColVideoLinks    = "lvideo"
	urlMetadataColAppLinks      = "lapp"
	urlMetadataColSnippet       = "snippet"
	urlMetadataColFavicon       = "favicon"
)

const urlMetadataTagSeparator = ","

// urlMetadataWireCodec translates between the url metadata domain type and
// YaCy's peer property-form row. It is the only thing outside this file that
// needs to know that row exists.
type urlMetadataWireCodec struct{}

func (urlMetadataWireCodec) decode(
	ctx context.Context,
	row string,
) (yacymodel.URLMetadata, error) {
	properties, err := propertyPairsOfRow(row)
	if err != nil {
		return yacymodel.URLMetadata{}, fmt.Errorf("%w: %w", yacymodel.ErrBadURLMetadata, err)
	}

	metadata, err := urlMetadataWireForm{properties: properties}.domain(ctx)
	if err != nil {
		return yacymodel.URLMetadata{}, fmt.Errorf("%w: %w", yacymodel.ErrBadURLMetadata, err)
	}

	return metadata, nil
}

func (urlMetadataWireCodec) encode(metadata yacymodel.URLMetadata) string {
	return urlMetadataWireFormFromDomain(metadata).row()
}

// urlMetadataWireForm is the url metadata in its property-form wire
// representation: a flat, protocol-defined column map.
type urlMetadataWireForm struct {
	properties map[string]string
	columns    []string
}

// domain projects the wire form onto the URLMetadata domain concept. A peer
// also sends hash, flags and score columns; the address is the identity, the
// flags restate other columns, and the score belongs to the sending peer's
// ranking, so none of them are read.
func (f urlMetadataWireForm) domain(ctx context.Context) (yacymodel.URLMetadata, error) {
	address, err := f.text(ctx, urlMetadataColAddress)
	if err != nil {
		return yacymodel.URLMetadata{}, fmt.Errorf("url metadata address: %w", err)
	}
	if address == "" {
		return yacymodel.URLMetadata{}, fmt.Errorf("url metadata address: empty")
	}

	texts, err := f.texts(
		ctx,
		urlMetadataColTitle,
		urlMetadataColAuthor,
		urlMetadataColTags,
		urlMetadataColPublisher,
		urlMetadataColMediaType,
		urlMetadataColSnippet,
		urlMetadataColFavicon,
	)
	if err != nil {
		return yacymodel.URLMetadata{}, err
	}

	location, err := f.location()
	if err != nil {
		return yacymodel.URLMetadata{}, fmt.Errorf("url metadata location: %w", err)
	}

	referrer, err := f.referrer()
	if err != nil {
		return yacymodel.URLMetadata{}, fmt.Errorf("url metadata referrer: %w", err)
	}

	return yacymodel.URLMetadata{
		Address:          address,
		Referrer:         referrer,
		Title:            texts[urlMetadataColTitle],
		Author:           texts[urlMetadataColAuthor],
		Tags:             splitTags(texts[urlMetadataColTags]),
		Publisher:        texts[urlMetadataColPublisher],
		Location:         location,
		Modified:         calendarDayWireCodec{}.decode(f.properties[urlMetadataColModified]),
		Loaded:           calendarDayWireCodec{}.decode(f.properties[urlMetadataColLoaded]),
		FreshUntil:       calendarDayWireCodec{}.decode(f.properties[urlMetadataColFreshUntil]),
		DocumentType:     f.documentType(),
		MediaType:        texts[urlMetadataColMediaType],
		Language:         f.language(),
		ByteSize:         f.cardinal(urlMetadataColByteSize),
		WordCount:        f.cardinal(urlMetadataColWordCount),
		LocalLinks:       f.cardinal(urlMetadataColLocalLinks),
		ExternalLinks:    f.cardinal(urlMetadataColExternalLinks),
		ImageLinks:       f.cardinal(urlMetadataColImageLinks),
		AudioLinks:       f.cardinal(urlMetadataColAudioLinks),
		VideoLinks:       f.cardinal(urlMetadataColVideoLinks),
		ApplicationLinks: f.cardinal(urlMetadataColAppLinks),
		Snippet:          texts[urlMetadataColSnippet],
		FaviconAddress:   texts[urlMetadataColFavicon],
	}, nil
}

func (f urlMetadataWireForm) text(ctx context.Context, column string) (string, error) {
	plain, err := decodeWireForm(ctx, f.properties[column])
	if err != nil {
		return "", fmt.Errorf("column %s: %w", column, err)
	}

	return plain, nil
}

func (f urlMetadataWireForm) texts(
	ctx context.Context,
	columns ...string,
) (map[string]string, error) {
	plain := make(map[string]string, len(columns))
	for _, column := range columns {
		value, err := f.text(ctx, column)
		if err != nil {
			return nil, fmt.Errorf("url metadata: %w", err)
		}
		plain[column] = value
	}

	return plain, nil
}

// cardinal reads a decimal column the way YaCy writes it, treating an absent or
// unreadable column as zero rather than rejecting the whole row.
func (f urlMetadataWireForm) cardinal(column string) int {
	value, ok := f.properties[column]
	if !ok {
		return 0
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}

	return number
}

func (f urlMetadataWireForm) documentType() yacymodel.DocumentType {
	value := f.properties[urlMetadataColDocType]
	if value == "" {
		return yacymodel.DocumentTypeText
	}

	return documentTypeByChar[value[0]]
}

// language keeps only the leading ISO 639-1 code and treats anything shorter as
// unknown: peers are known to send three-letter codes and empty columns alike.
func (f urlMetadataWireForm) language() yacymodel.Optional[yacymodel.Language] {
	value := f.properties[urlMetadataColLanguage]
	if len(value) < yacymodel.LanguageCodeLength {
		return yacymodel.None[yacymodel.Language]()
	}
	language, err := yacymodel.ParseLanguage(value[:yacymodel.LanguageCodeLength])
	if err != nil {
		return yacymodel.None[yacymodel.Language]()
	}

	return yacymodel.Some(language)
}

// location treats the origin as absent, the way YaCy treats a zero pair as
// carrying no location metadata.
func (f urlMetadataWireForm) location() (yacymodel.Optional[yacymodel.Coordinates], error) {
	latitude := f.degrees(urlMetadataColLatitude)
	longitude := f.degrees(urlMetadataColLongitude)
	if latitude == 0 && longitude == 0 {
		return yacymodel.None[yacymodel.Coordinates](), nil
	}

	coordinates, err := yacymodel.NewCoordinates(latitude, longitude)
	if err != nil {
		return yacymodel.None[yacymodel.Coordinates](), err
	}

	return yacymodel.Some(coordinates), nil
}

func (f urlMetadataWireForm) degrees(column string) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(f.properties[column]), 64)
	if err != nil {
		return 0
	}

	return value
}

func (f urlMetadataWireForm) referrer() (yacymodel.Optional[yacymodel.URLHash], error) {
	value := f.properties[urlMetadataColReferrer]
	if value == "" {
		return yacymodel.None[yacymodel.URLHash](), nil
	}

	referrer, err := yacymodel.ParseURLHash(value)
	if err != nil {
		return yacymodel.None[yacymodel.URLHash](), err
	}

	return yacymodel.Some(referrer), nil
}

func splitTags(joined string) []string {
	if joined == "" {
		return nil
	}

	return strings.Split(joined, urlMetadataTagSeparator)
}

// urlMetadataWireFormFromDomain builds the wire form for domain metadata,
// emitting the columns in the order YaCy itself writes them. The hash column
// is recomputed from the address, which is what a receiving peer does too.
func urlMetadataWireFormFromDomain(m yacymodel.URLMetadata) urlMetadataWireForm {
	f := urlMetadataWireForm{properties: map[string]string{}}

	hash, err := m.Hash()
	if err == nil {
		f.put(urlMetadataColHash, hash.String())
	}
	f.putEncoded(urlMetadataColAddress, m.Address)
	f.putEncoded(urlMetadataColTitle, m.Title)
	f.putEncoded(urlMetadataColAuthor, m.Author)
	f.putEncoded(urlMetadataColTags, strings.Join(m.Tags, urlMetadataTagSeparator))
	f.putEncoded(urlMetadataColPublisher, m.Publisher)
	f.putDegrees(m.Location)
	f.putDay(urlMetadataColModified, m.Modified)
	f.putDay(urlMetadataColLoaded, m.Loaded)
	f.putDay(urlMetadataColFreshUntil, m.FreshUntil)
	f.putReferrer(m.Referrer)
	f.put(urlMetadataColByteSize, strconv.Itoa(m.ByteSize))
	f.put(urlMetadataColWordCount, strconv.Itoa(m.WordCount))
	f.put(urlMetadataColDocType, string(charByDocumentType[m.DocumentType]))
	if m.MediaType != "" {
		f.putEncoded(urlMetadataColMediaType, m.MediaType)
	}
	f.put(urlMetadataColFlags, yacymodel.Encode(urlMetadataFlags(m)))
	if language, ok := m.Language.Get(); ok {
		f.put(urlMetadataColLanguage, language.String())
	}
	f.put(urlMetadataColLocalLinks, strconv.Itoa(m.LocalLinks))
	f.put(urlMetadataColExternalLinks, strconv.Itoa(m.ExternalLinks))
	f.put(urlMetadataColImageLinks, strconv.Itoa(m.ImageLinks))
	f.put(urlMetadataColAudioLinks, strconv.Itoa(m.AudioLinks))
	f.put(urlMetadataColVideoLinks, strconv.Itoa(m.VideoLinks))
	f.put(urlMetadataColAppLinks, strconv.Itoa(m.ApplicationLinks))
	if m.Snippet != "" {
		f.putEncoded(urlMetadataColSnippet, m.Snippet)
	}
	if m.FaviconAddress != "" {
		f.putEncoded(urlMetadataColFavicon, m.FaviconAddress)
	}

	return f
}

func urlMetadataFlags(m yacymodel.URLMetadata) bitfield {
	flags := make(bitfield, appearanceFlagsByteWidth)
	flags.setBit(appearanceFlagBitIndexOf, m.IsDirectoryListing())
	flags.setBit(appearanceFlagBitHasLocation, m.HasLocation())
	flags.setBit(appearanceFlagBitHasImage, m.HasImage())
	flags.setBit(appearanceFlagBitHasAudio, m.HasAudio())
	flags.setBit(appearanceFlagBitHasVideo, m.HasVideo())
	flags.setBit(appearanceFlagBitHasApp, m.HasApplication())

	return flags
}

func (f *urlMetadataWireForm) put(column, value string) {
	f.properties[column] = value
	f.columns = append(f.columns, column)
}

func (f *urlMetadataWireForm) putEncoded(column, value string) {
	f.put(column, encodeBase64WireForm(value))
}

func (f *urlMetadataWireForm) putDay(column string, day yacymodel.Optional[yacymodel.CalendarDay]) {
	f.put(column, calendarDayWireCodec{}.encode(day))
}

func (f *urlMetadataWireForm) putDegrees(location yacymodel.Optional[yacymodel.Coordinates]) {
	coordinates, _ := location.Get()
	f.put(urlMetadataColLatitude, strconv.FormatFloat(coordinates.Latitude, 'f', -1, 64))
	f.put(urlMetadataColLongitude, strconv.FormatFloat(coordinates.Longitude, 'f', -1, 64))
}

func (f *urlMetadataWireForm) putReferrer(referrer yacymodel.Optional[yacymodel.URLHash]) {
	if value, ok := referrer.Get(); ok {
		f.put(urlMetadataColReferrer, value.String())
		return
	}
	f.put(urlMetadataColReferrer, "")
}

func (f urlMetadataWireForm) row() string {
	var b strings.Builder
	b.WriteByte(propertyOpen)
	for i, column := range f.columns {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(column)
		b.WriteByte('=')
		b.WriteString(f.properties[column])
	}
	b.WriteByte(propertyClose)

	return b.String()
}
