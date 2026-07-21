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
	originator  Hash
	category    NewsCategory
	created     time.Time
	received    Optional[time.Time]
	distributed int
	attributes  map[string]string
}

//nolint:revive // a news record is built from all its fields
func NewPeerNews(
	originator Hash,
	category NewsCategory,
	created time.Time,
	received Optional[time.Time],
	distributed int,
	attributes map[string]string,
) (PeerNews, error) {
	if originator.IsZero() {
		return PeerNews{}, fmt.Errorf("%w: missing originator", ErrBadPeerNews)
	}
	if category.IsZero() {
		return PeerNews{}, fmt.Errorf("%w: missing category", ErrBadPeerNews)
	}
	return PeerNews{
		originator:  originator,
		category:    category,
		created:     created,
		received:    received,
		distributed: distributed,
		attributes:  attributes,
	}, nil
}

func (n PeerNews) Originator() Hash { return n.originator }

func (n PeerNews) Category() NewsCategory { return n.category }

func (n PeerNews) Created() time.Time { return n.created }

func (n PeerNews) Received() Optional[time.Time] { return n.received }

func (n PeerNews) Distributed() int { return n.distributed }

func (n PeerNews) Attributes() map[string]string { return n.attributes }
