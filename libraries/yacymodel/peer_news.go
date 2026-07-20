package yacymodel

import (
	"errors"
	"fmt"
	"time"
)

const newsAttributesMaxLength = 974

var ErrBadPeerNews = errors.New("bad peer news")

// PeerNews is a single event a peer gossips to the network, piggybacked on its
// seed. It is YaCy's decoded NewsDB record.
type PeerNews struct {
	Originator  Hash
	Category    NewsCategory
	Created     time.Time
	Received    Optional[time.Time]
	Distributed int
	Attributes  map[string]string
}

func (n PeerNews) Validate() error {
	if !n.Originator.Valid() {
		return fmt.Errorf("%w: originator %q", ErrBadPeerNews, n.Originator)
	}
	if _, err := ParseNewsCategory(string(n.Category)); err != nil {
		return fmt.Errorf("%w: %w", ErrBadPeerNews, err)
	}
	total := 0
	for key, value := range n.Attributes {
		total += len(key) + len(value)
	}
	if total > newsAttributesMaxLength {
		return fmt.Errorf("%w: attributes %d bytes", ErrBadPeerNews, total)
	}
	return nil
}
