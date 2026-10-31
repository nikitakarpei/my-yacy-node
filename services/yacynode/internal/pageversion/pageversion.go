// Package pageversion tells versions of a page apart, orders them, and says
// whether an arrived version joins a stored one. The page fields that a crawl
// writes into its postings and its URL metadata tell versions apart. A version
// also knows its source: this node or the network of the peer that delivered
// it. The source does not tell versions apart.
package pageversion

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
)

type Version struct {
	fingerprint pagefingerprint.Fingerprint
	source      Source
}

type Joining byte

const (
	JoiningAllowed Joining = iota + 1
	JoiningRefusedOtherVersion
	JoiningRefusedOtherSource
)

func FromMetadata(metadata yacymodel.URLMetadata, source Source) Version {
	return Version{fingerprint: pagefingerprint.FromMetadata(metadata), source: source}
}

func FromPosting(posting yacymodel.RWIPosting, source Source) Version {
	return Version{fingerprint: pagefingerprint.FromPosting(posting), source: source}
}

func (v Version) Is(other Version) bool {
	return v.fingerprint == other.fingerprint
}

func (v Version) IsReplacedBy(arrived Version) bool {
	return v.fingerprint.Modified < arrived.fingerprint.Modified
}

func (v Version) IsDatedAfter(now time.Time) bool {
	today, inRange := yacymodel.MicroDateOf(now).Get()

	return inRange && v.fingerprint.Modified > today
}

func (v Version) JoiningOf(arrived Version) Joining {
	switch {
	case !v.Is(arrived):
		return JoiningRefusedOtherVersion
	case v.source != arrived.source:
		return JoiningRefusedOtherSource
	default:
		return JoiningAllowed
	}
}
