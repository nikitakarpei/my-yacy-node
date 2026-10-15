package urlpageversion

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

const futureDaysAccepted = 1

type indexedVersion struct {
	Version yacymodel.MicroDate
	Origin  rwiorigin.Origin
}

type PageVersions struct {
	indexed  *vault.Collection[yacymodel.URLHash, indexedVersion]
	postings Postings
	now      func() time.Time
	swaps    *swapAllowance
	observer Observer
}

func (v *PageVersions) Admit(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	pageVersion yacymodel.MicroDate,
) (Verdict, error) {
	verdict, err := v.verdictFor(tx, origin, posting, pageVersion)
	if err != nil {
		return "", err
	}
	v.observer.ObserveVerdict(tx, verdict)

	return verdict, nil
}

func (v *PageVersions) verdictFor(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	pageVersion yacymodel.MicroDate,
) (Verdict, error) {
	if posting.PageVersion() > v.latestAcceptedVersion() {
		return VerdictFutureDate, nil
	}
	indexed, found, err := v.indexed.Get(tx, posting.URLHash)
	if err != nil {
		return "", fmt.Errorf("read indexed page version: %w", err)
	}
	metadataIsOlder := pageVersion < posting.PageVersion()

	switch {
	case !found && metadataIsOlder:
		return VerdictNewerThanMetadata, nil
	case !found:
		return VerdictNewURL, v.replaceWith(tx, origin, posting)
	case posting.PageVersion() < indexed.Version:
		return VerdictOlderVersion, nil
	case posting.PageVersion() == indexed.Version:
		return v.fillIn(tx, origin, posting, indexed)
	case metadataIsOlder:
		return VerdictNewerThanMetadata, nil
	default:
		return v.swap(tx, origin, posting)
	}
}

func (v *PageVersions) latestAcceptedVersion() yacymodel.MicroDate {
	return yacymodel.MicroDateFromTime(v.now()) + futureDaysAccepted
}

func (v *PageVersions) replaceWith(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) error {
	if err := v.postings.Admitter.Admit(tx, posting); err != nil {
		return fmt.Errorf("admit posting: %w", err)
	}
	if _, err := v.indexed.Put(tx, posting.URLHash, indexedVersion{
		Version: posting.PageVersion(),
		Origin:  origin,
	}); err != nil {
		return fmt.Errorf("store indexed page version: %w", err)
	}

	return nil
}

func (v *PageVersions) fillIn(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	indexed indexedVersion,
) (Verdict, error) {
	if origin != indexed.Origin {
		return VerdictSameVersion, nil
	}
	_, stored, err := v.postings.Index.PostingOf(tx, posting.WordHash, posting.URLHash)
	if err != nil {
		return "", fmt.Errorf("read indexed posting: %w", err)
	}
	if stored {
		return VerdictSameVersion, nil
	}
	if err := v.postings.Admitter.Admit(tx, posting); err != nil {
		return "", fmt.Errorf("admit missing posting: %w", err)
	}

	return VerdictMissingWord, nil
}

func (v *PageVersions) swap(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
) (Verdict, error) {
	indexedURLs, err := v.indexed.Len(tx)
	if err != nil {
		return "", fmt.Errorf("count indexed page versions: %w", err)
	}
	if !v.swaps.allows(origin.Network, indexedURLs) {
		return VerdictSwapLimit, nil
	}
	tx.RunAfterCommit(func() { v.swaps.count(origin.Network) })
	if err := v.purgePostingsOf(tx, posting.URLHash); err != nil {
		return "", err
	}

	return VerdictNewerVersion, v.replaceWith(tx, origin, posting)
}

func (v *PageVersions) purgePostingsOf(tx *vault.Txn, url yacymodel.URLHash) error {
	words, err := v.postings.References.WordsReferencing(tx, url)
	if err != nil {
		return fmt.Errorf("words of the older page version: %w", err)
	}
	for _, word := range words {
		if _, err := v.postings.Purger.PurgePosting(tx, word, url); err != nil {
			return fmt.Errorf("purge posting of the older page version: %w", err)
		}
	}

	return nil
}

func (v *PageVersions) URLStored(*vault.Txn, yacymodel.URLHash, yacymodel.MicroDate) error {
	return nil
}

func (v *PageVersions) URLPurged(tx *vault.Txn, url yacymodel.URLHash) error {
	if _, err := v.indexed.Delete(tx, url); err != nil {
		return fmt.Errorf("forget indexed page version: %w", err)
	}

	return nil
}
