package yacyproto

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"log/slog"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type URLMetadataRequest struct {
	NetworkName string
	URLs        []yacymodel.URLHash
}

type URLMetadataResponse struct {
	URLs []yacymodel.URLMetadata
}

func (r URLMetadataRequest) Form() url.Values {
	form := url.Values{}
	putString(form, FieldNetworkName, r.NetworkName)
	putString(form, FieldCall, CallURLHashList)
	putString(form, FieldHashes, concatURLHashes(r.URLs))

	return form
}

func ParseURLMetadataResponse(ctx context.Context, body []byte) (URLMetadataResponse, error) {
	feed, err := urlMetadataFeedOf(body)
	if err != nil {
		return URLMetadataResponse{}, err
	}

	if feed.Peer.Response != urlMetadataFeedAccepted {
		return URLMetadataResponse{}, fmt.Errorf(
			"urls response: %w: %s %q",
			ErrBadField,
			FieldResponse,
			feed.Peer.Response,
		)
	}

	return URLMetadataResponse{URLs: urlMetadataOf(ctx, feed.Channel.Items)}, nil
}

// urlMetadataFeedOf reads the feed leniently. A peer writes the title and the
// author of a document into the feed without an XML escape, so an ampersand in
// either arrives raw, and a lenient reader keeps such a sequence as it stands.
func urlMetadataFeedOf(body []byte) (urlMetadataFeed, error) {
	reader := xml.NewDecoder(bytes.NewReader(body))
	reader.Strict = false

	var feed urlMetadataFeed
	if err := reader.Decode(&feed); err != nil {
		return urlMetadataFeed{}, fmt.Errorf("urls response: %w", err)
	}

	return feed, nil
}

const urlMetadataFeedAccepted = "ok"

type urlMetadataFeed struct {
	Peer    urlMetadataFeedPeer    `xml:"yacy"`
	Channel urlMetadataFeedChannel `xml:"channel"`
}

type urlMetadataFeedPeer struct {
	Response string `xml:"response"`
}

type urlMetadataFeedChannel struct {
	Items []urlMetadataFeedItem `xml:"item"`
}

type urlMetadataFeedItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	Author  string `xml:"author"`
	PubDate string `xml:"pubDate"`
	GUID    string `xml:"guid"`
}

func urlMetadataOf(ctx context.Context, items []urlMetadataFeedItem) []yacymodel.URLMetadata {
	urls := make([]yacymodel.URLMetadata, 0, len(items))
	for _, item := range items {
		metadata, err := item.domain()
		if err != nil {
			slog.WarnContext(
				ctx,
				"url metadata item discarded",
				slog.String("reason", "parse failed"),
				slog.String("guid", item.GUID),
				slog.Any("error", err),
			)

			continue
		}

		urls = append(urls, metadata)
	}

	return urls
}

// domain projects the feed item onto the URLMetadata domain concept. A peer also
// sends a referrer and a description; the referrer arrives as an address where
// the domain concept holds a name, and the description repeats the title, so
// neither is read.
func (i urlMetadataFeedItem) domain() (yacymodel.URLMetadata, error) {
	hash, err := yacymodel.ParseURLHash(i.GUID)
	if err != nil {
		return yacymodel.URLMetadata{}, fmt.Errorf("url metadata guid: %w", err)
	}

	return yacymodel.URLMetadata{
		Hash:     hash,
		Address:  i.Link,
		Title:    i.Title,
		Author:   i.Author,
		Modified: calendarDayOfInstantText(i.PubDate),
	}, nil
}

func calendarDayOfInstantText(text string) yacymodel.Optional[yacymodel.CalendarDay] {
	instant, ok := instantWireCodec{}.decode(text)
	if !ok {
		return yacymodel.None[yacymodel.CalendarDay]()
	}

	return yacymodel.Some(yacymodel.CalendarDayOf(instant))
}
