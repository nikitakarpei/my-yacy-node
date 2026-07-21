package yacyproto

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	newsColOriginator  = "ori"
	newsColCategory    = "cat"
	newsColCreated     = "cre"
	newsColReceived    = "rec"
	newsColDistributed = "dis"

	newsRecordMaxLength = 1024
	newsMapSeparator    = ", "
)

// peerNewsWireCodec translates between the peer news domain type and YaCy's news
// field: a base64-framed Java map string carrying the originator, category,
// dates, distribution counter, and per-category gossip attributes.
type peerNewsWireCodec struct{}

func (peerNewsWireCodec) decode(ctx context.Context, raw string) (yacymodel.PeerNews, error) {
	plain, err := decodeWireForm(ctx, raw)
	if err != nil {
		return yacymodel.PeerNews{}, fmt.Errorf("%w: %w", yacymodel.ErrBadPeerNews, err)
	}
	if len(plain) > newsRecordMaxLength {
		return yacymodel.PeerNews{}, fmt.Errorf(
			"%w: record %d bytes exceeds %d",
			yacymodel.ErrBadPeerNews, len(plain), newsRecordMaxLength,
		)
	}
	if open := strings.IndexByte(plain, '{'); open >= 0 {
		plain = plain[open+1:]
	}
	if end := strings.LastIndexByte(plain, '}'); end >= 0 {
		plain = plain[:end]
	}
	fields, err := parsePropertyPairs(plain)
	if err != nil {
		return yacymodel.PeerNews{}, fmt.Errorf("%w: %w", yacymodel.ErrBadPeerNews, err)
	}

	originator, err := yacymodel.ParseHash(fields[newsColOriginator])
	if err != nil {
		return yacymodel.PeerNews{}, fmt.Errorf("%w: originator: %w", yacymodel.ErrBadPeerNews, err)
	}
	category, err := yacymodel.ParseNewsCategory(fields[newsColCategory])
	if err != nil {
		return yacymodel.PeerNews{}, fmt.Errorf("%w: %w", yacymodel.ErrBadPeerNews, err)
	}
	created, ok := seedTimestampWireCodec{}.decode(fields[newsColCreated])
	if !ok {
		return yacymodel.PeerNews{}, fmt.Errorf(
			"%w: created %q", yacymodel.ErrBadPeerNews, fields[newsColCreated],
		)
	}

	news := yacymodel.PeerNews{
		Originator:  originator,
		Category:    category,
		Created:     created,
		Distributed: newsDistributed(fields[newsColDistributed]),
		Attributes:  newsAttributes(fields),
	}
	if received, ok := (seedTimestampWireCodec{}).decode(fields[newsColReceived]); ok {
		news.Received = yacymodel.Some(received)
	}
	if err := news.Validate(); err != nil {
		return yacymodel.PeerNews{}, err
	}

	return news, nil
}

func newsDistributed(text string) int {
	distributed, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0
	}
	return distributed
}

func newsAttributes(fields map[string]string) map[string]string {
	standards := map[string]bool{
		newsColOriginator: true, newsColCategory: true, newsColCreated: true,
		newsColReceived: true, newsColDistributed: true,
	}
	attributes := map[string]string{}
	for key, value := range fields {
		if standards[key] {
			continue
		}
		attributes[key] = value
	}
	return attributes
}

func (peerNewsWireCodec) encode(news yacymodel.PeerNews) string {
	timestamp := seedTimestampWireCodec{}
	pairs := []string{
		newsColOriginator + "=" + news.Originator.String(),
		newsColCategory + "=" + news.Category.String(),
		newsColCreated + "=" + timestamp.encode(news.Created),
	}
	if received, ok := news.Received.Get(); ok {
		pairs = append(pairs, newsColReceived+"="+timestamp.encode(received))
	}
	pairs = append(pairs, newsColDistributed+"="+strconv.Itoa(news.Distributed))
	pairs = append(pairs, sortedAttributePairs(news.Attributes)...)

	return encodeBase64WireForm("{" + strings.Join(pairs, newsMapSeparator) + "}")
}

func sortedAttributePairs(attributes map[string]string) []string {
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+"="+attributes[key])
	}
	return pairs
}
