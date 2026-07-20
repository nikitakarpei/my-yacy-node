package yacyproto

import (
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	peerTagsSeparator = "|"
	peerTagsWildcard  = "*"
)

// peerTagsWireCodec translates between the peer tags domain type and YaCy's
// bar-separated tag field, where a lone wildcard means match all topics.
type peerTagsWireCodec struct{}

func (peerTagsWireCodec) decode(text string) (yacymodel.PeerTags, error) {
	if text == "" || text == peerTagsWildcard {
		return yacymodel.MatchAllTags(), nil
	}
	var tags []yacymodel.Tag
	for _, field := range strings.Split(text, peerTagsSeparator) {
		if field == "" {
			continue
		}
		tag, err := yacymodel.ParseTag(field)
		if err != nil {
			return yacymodel.PeerTags{}, err
		}
		tags = append(tags, tag)
	}
	return yacymodel.NewPeerTags(tags)
}

func (peerTagsWireCodec) encode(tags yacymodel.PeerTags) string {
	if tags.MatchesAll() {
		return peerTagsWildcard
	}
	fields := make([]string, 0, len(tags.Tags()))
	for _, tag := range tags.Tags() {
		fields = append(fields, tag.String())
	}
	return strings.Join(fields, peerTagsSeparator)
}
