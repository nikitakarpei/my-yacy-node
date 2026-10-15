// Package urlpageversion keeps one page version per URL for the postings peers
// send: the version of the postings indexed and their origin. It rejects older
// postings, takes missing words of the indexed version only from its origin, and
// swaps a URL to a newer version once the URL metadata is no older.
package urlpageversion

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Verdict string

const (
	VerdictNewURL            Verdict = "new_url"
	VerdictMissingWord       Verdict = "missing_word"
	VerdictNewerVersion      Verdict = "newer_version"
	VerdictNewerThanMetadata Verdict = "newer_than_metadata"
	VerdictFutureDate        Verdict = "future_date"
	VerdictOlderVersion      Verdict = "older_version"
	VerdictSameVersion       Verdict = "same_version"
	VerdictSwapLimit         Verdict = "swap_limit"
)

type Outcome string

const (
	OutcomeIndexed Outcome = "indexed"
	OutcomeHeld    Outcome = "held"
	OutcomeDropped Outcome = "dropped"
)

func (v Verdict) Outcome() Outcome {
	switch v {
	case VerdictNewURL, VerdictMissingWord, VerdictNewerVersion:
		return OutcomeIndexed
	case VerdictNewerThanMetadata:
		return OutcomeHeld
	default:
		return OutcomeDropped
	}
}

type Observer interface {
	ObserveVerdict(tx *vault.Txn, verdict Verdict)
}

type Config struct {
	SwapFloorPerHour int
	SwapSharePerHour float64
	Observer         Observer
}

type Postings struct {
	Index      rwipostings.PostingIndex
	References rwipostings.URLReferences
	Admitter   rwipostings.PostingAdmitter
	Purger     rwipostings.PostingPurger
}

func Open(
	v *vault.Vault,
	postings Postings,
	now func() time.Time,
	config Config,
) (*PageVersions, error) {
	indexed, err := registerIndexedVersions(v)
	if err != nil {
		return nil, err
	}

	return &PageVersions{
		indexed:  indexed,
		postings: postings,
		now:      now,
		swaps:    newSwapAllowance(now, config.SwapFloorPerHour, config.SwapSharePerHour),
		observer: config.Observer,
	}, nil
}
