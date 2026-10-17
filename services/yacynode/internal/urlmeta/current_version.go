package urlmeta

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

type CurrentVersion struct {
	Fingerprint pagefingerprint.Fingerprint
	Source      Source
}

func currentVersionOf(stored storedURLMetadata) CurrentVersion {
	return CurrentVersion{
		Fingerprint: pagefingerprint.FromMetadata(stored.metadata),
		Source:      stored.source,
	}
}

func (v CurrentVersion) Describes(fingerprint pagefingerprint.Fingerprint) bool {
	return v.Fingerprint == fingerprint
}

func (v CurrentVersion) IsOlderThan(fingerprint pagefingerprint.Fingerprint) bool {
	return v.Fingerprint.Modified < fingerprint.Modified
}

func (v CurrentVersion) CameFrom(origin rwiorigin.Origin) bool {
	return v.Source == SourcePeer(origin)
}
