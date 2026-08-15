package yacymodel

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrBadURLMetadata reports url metadata that cannot be read back into the
// domain type, whether it came off the peer wire or off local disk.
var ErrBadURLMetadata = errors.New("bad url metadata")

const directoryListingTag = "indexof"

const (
	imageMediaTypePrefix       = "image/"
	audioMediaTypePrefix       = "audio/"
	videoMediaTypePrefix       = "video/"
	applicationMediaTypePrefix = "application/"
)

// URLMetadata is what the index knows about the document at one address.
type URLMetadata struct {
	Address          string
	Referrer         Optional[URLHash]
	Title            string
	Author           string
	Tags             []string
	Publisher        string
	Location         Optional[Coordinates]
	Modified         Optional[CalendarDay]
	Loaded           Optional[CalendarDay]
	FreshUntil       Optional[CalendarDay]
	DocumentType     DocumentType
	MediaType        string
	Language         Optional[Language]
	ByteSize         int
	WordCount        int
	LocalLinks       int
	ExternalLinks    int
	ImageLinks       int
	AudioLinks       int
	VideoLinks       int
	ApplicationLinks int
	Snippet          string
	FaviconAddress   string
}

func (m URLMetadata) Hash() (URLHash, error) {
	address, err := url.Parse(m.Address)
	if err != nil {
		return URLHash{}, fmt.Errorf("%w: address %q: %w", ErrBadURLMetadata, m.Address, err)
	}

	return URLNormalformOf(address).Hash(), nil
}

// Freshness is the day this metadata last stood for the document, preferring
// the most recently established of the three days the document carries.
func (m URLMetadata) Freshness() Optional[CalendarDay] {
	for _, day := range []Optional[CalendarDay]{m.Loaded, m.Modified, m.FreshUntil} {
		if day.Present() {
			return day
		}
	}

	return None[CalendarDay]()
}

func (m URLMetadata) IsDirectoryListing() bool {
	for _, tag := range m.Tags {
		if strings.Contains(tag, directoryListingTag) {
			return true
		}
	}

	return false
}

func (m URLMetadata) HasLocation() bool {
	return m.Location.Present()
}

// TODO: YaCy's getContentDomain() also classifies by file extension when the
// media type is unhelpful; only the media-type path is ported.
func (m URLMetadata) HasImage() bool {
	return m.ImageLinks > 0 || strings.HasPrefix(m.MediaType, imageMediaTypePrefix)
}

func (m URLMetadata) HasAudio() bool {
	return m.AudioLinks > 0 || strings.HasPrefix(m.MediaType, audioMediaTypePrefix)
}

func (m URLMetadata) HasVideo() bool {
	return m.VideoLinks > 0 || strings.HasPrefix(m.MediaType, videoMediaTypePrefix)
}

func (m URLMetadata) HasApplication() bool {
	return m.ApplicationLinks > 0 ||
		strings.HasPrefix(m.MediaType, applicationMediaTypePrefix)
}
