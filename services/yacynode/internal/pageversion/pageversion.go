// Package pageversion tells versions of a page apart, orders them, and says
// whether an arrived version joins a stored one. The page fields that a crawl
// writes into its postings and its URL metadata, and the postings root when
// the crawl made one, tell versions apart. A version also knows its source:
// this node or the network of the peer that delivered it.
package pageversion

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
)

type Version struct {
	fingerprint pagefingerprint.Fingerprint
	root        yacymodel.Optional[yacymodel.PostingsRoot]
	source      Source
}

type Replacement byte

const (
	ReplacementAllowedNewerVersion Replacement = iota + 1
	ReplacementAllowedRootedVersion
	ReplacementRefusedNotNewerVersion
	ReplacementRefusedUnrootedVersion
)

type Joining byte

const (
	JoiningAllowed Joining = iota + 1
	JoiningRefusedOtherVersion
	JoiningRefusedOtherSource
	JoiningRefusedUnrootedVersion
)

func FromMetadata(metadata yacymodel.URLMetadata, source Source) Version {
	return Version{
		fingerprint: pagefingerprint.FromMetadata(metadata),
		root:        metadata.PostingsRoot,
		source:      source,
	}
}

func FromPagePosting(pagePosting yacymodel.PagePosting, source Source) Version {
	return Version{
		fingerprint: pagefingerprint.FromPosting(pagePosting.Posting()),
		root:        pagePosting.Root(),
		source:      source,
	}
}

func (v Version) Is(other Version) bool {
	return v.fingerprint == other.fingerprint && v.root == other.root
}

func (v Version) IsReplacedBy(arrived Version) bool {
	return v.ReplacementBy(arrived).IsAllowed()
}

func (v Version) ReplacementBy(arrived Version) Replacement {
	switch {
	case v.root.Present() && !arrived.root.Present():
		return ReplacementRefusedUnrootedVersion
	case !v.root.Present() && arrived.root.Present():
		return ReplacementAllowedRootedVersion
	case v.fingerprint.Modified < arrived.fingerprint.Modified:
		return ReplacementAllowedNewerVersion
	default:
		return ReplacementRefusedNotNewerVersion
	}
}

func (r Replacement) IsAllowed() bool {
	return r == ReplacementAllowedNewerVersion || r == ReplacementAllowedRootedVersion
}

func (v Version) IsDatedAfter(now time.Time) bool {
	today, inRange := yacymodel.MicroDateOf(now).Get()

	return inRange && v.fingerprint.Modified > today
}

func (v Version) JoiningOf(arrived Version) Joining {
	switch {
	case v.Is(arrived) && (v.root.Present() || v.source == arrived.source):
		return JoiningAllowed
	case v.Is(arrived):
		return JoiningRefusedOtherSource
	case v.root.Present() && !arrived.root.Present():
		return JoiningRefusedUnrootedVersion
	default:
		return JoiningRefusedOtherVersion
	}
}
