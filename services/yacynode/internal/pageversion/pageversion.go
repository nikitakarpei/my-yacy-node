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
	ReplacementRefused Replacement = iota + 1
	ReplacementAllowedOlderVersion
	ReplacementAllowedUnrootedVersion
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

func FromPosting(posting yacymodel.RWIPosting, source Source) Version {
	return Version{
		fingerprint: pagefingerprint.FromPosting(posting),
		root:        yacymodel.None[yacymodel.PostingsRoot](),
		source:      source,
	}
}

func FromProvenPosting(provenPosting yacymodel.ProvenPosting, source Source) Version {
	return Version{
		fingerprint: pagefingerprint.FromPosting(provenPosting.Posting()),
		root:        yacymodel.Some(provenPosting.Root()),
		source:      source,
	}
}

func (v Version) Is(other Version) bool {
	return v.fingerprint == other.fingerprint && v.root == other.root
}

func (v Version) IsReplacedBy(arrived Version) bool {
	return v.ReplacementBy(arrived) != ReplacementRefused
}

func (v Version) ReplacementBy(arrived Version) Replacement {
	switch {
	case v.fingerprint.Modified < arrived.fingerprint.Modified:
		return ReplacementAllowedOlderVersion
	case v.fingerprint.Modified == arrived.fingerprint.Modified &&
		!v.root.Present() && arrived.root.Present():
		return ReplacementAllowedUnrootedVersion
	default:
		return ReplacementRefused
	}
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
	case v.fingerprint == arrived.fingerprint && v.root.Present() && !arrived.root.Present():
		return JoiningRefusedUnrootedVersion
	default:
		return JoiningRefusedOtherVersion
	}
}
