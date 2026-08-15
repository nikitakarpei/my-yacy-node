package yacymodel

import (
	"errors"
	"fmt"
	"strings"
)

const (
	MaxTagLength   = 100
	MaxTagsPerPeer = 64
)

var (
	ErrBadTag      = errors.New("bad tag")
	ErrBadPeerTags = errors.New("bad peer tags")
)

// Tag is a single self-assigned peer topic label.
type Tag struct {
	value string
}

func ParseTag(s string) (Tag, error) {
	if s == "" || len(s) > MaxTagLength {
		return Tag{}, fmt.Errorf("%w: length %d", ErrBadTag, len(s))
	}
	if strings.ContainsAny(s, ",=|{}") {
		return Tag{}, fmt.Errorf("%w: %q", ErrBadTag, s)
	}
	return Tag{value: s}, nil
}

func (t Tag) String() string { return t.value }

// PeerTags is a peer's set of topic labels. The zero value matches all topics,
// mirroring YaCy's wildcard default.
type PeerTags struct {
	tags []Tag
}

func MatchAllTags() PeerTags { return PeerTags{} }

func NewPeerTags(tags []Tag) (PeerTags, error) {
	if len(tags) > MaxTagsPerPeer {
		return PeerTags{}, fmt.Errorf("%w: %d tags", ErrBadPeerTags, len(tags))
	}
	return PeerTags{tags: tags}, nil
}

func (p PeerTags) MatchesAll() bool { return len(p.tags) == 0 }

func (p PeerTags) Tags() []Tag { return p.tags }
