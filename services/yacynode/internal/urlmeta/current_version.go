package urlmeta

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"

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
