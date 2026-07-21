package yacymodel

import (
	"errors"
	"fmt"
	"time"
)

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
	if n.Originator.IsZero() {
		return fmt.Errorf("%w: missing originator", ErrBadPeerNews)
	}
	if n.Category.IsZero() {
		return fmt.Errorf("%w: missing category", ErrBadPeerNews)
	}
	return nil
}
