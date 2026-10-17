// Package rwiadmission decides the fate of every inbound peer posting, when a
// peer sends it and when URL metadata of its URL is stored. A posting is dropped,
// waits in escrow, or joins the index against the page version indexed for its
// URL. The receipt names the URLs of waiting postings so the sender can send
// their metadata.
package rwiadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiindexedversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type PostingReceiver interface {
	Receive(
		ctx context.Context,
		origin rwiorigin.Origin,
		postings []yacymodel.RWIPosting,
	) (Receipt, error)
}

type Escrow interface {
	AtCapacity(ctx context.Context) (bool, error)
	Hold(tx *vault.Txn, origin rwiorigin.Origin, posting yacymodel.RWIPosting) error
	PostingsWaitingFor(tx *vault.Txn, url yacymodel.URLHash) ([]rwiescrow.WaitingPosting, error)
	Release(tx *vault.Txn, posting yacymodel.RWIPosting) error
}

type IndexedVersions interface {
	Of(tx *vault.Txn, url yacymodel.URLHash) (rwiindexedversion.IndexedVersion, bool, error)
	Record(tx *vault.Txn, url yacymodel.URLHash, version rwiindexedversion.IndexedVersion) error
	Count(tx *vault.Txn) (int, error)
}

type Stopwords interface {
	Contains(word yacymodel.Hash, language yacymodel.Language) bool
}

type Observer interface {
	ObserveOutcome(outcome Outcome, postings int)
}

type Receipt struct {
	Busy       bool
	Pause      time.Duration
	UnknownURL []yacymodel.URLHash
}

type Config struct {
	PostingCap       int
	Pause            time.Duration
	SwapFloorPerHour int
	SwapSharePerHour float64
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	v *vault.Vault,
	urls urlmeta.URLDirectory,
	escrow Escrow,
	indexedVersions IndexedVersions,
	postings rwipostings.PostingIndex,
	references rwipostings.URLReferences,
	admitter rwipostings.PostingAdmitter,
	purger rwipostings.PostingPurger,
	stopwords Stopwords,
	observer Observer,
	now func() time.Time,
	config Config,
) PostingAdmission {
	swapAllowance := newSwapAllowance(
		indexedVersions,
		now,
		config.SwapFloorPerHour,
		config.SwapSharePerHour,
	)

	return PostingAdmission{
		vault:  v,
		urls:   urls,
		escrow: escrow,
		judge: postingJudge{
			indexedVersions: indexedVersions,
			postings:        postings,
			swapAllowance:   swapAllowance,
			now:             now,
		},
		indexedVersions: indexedVersions,
		references:      references,
		admitter:        admitter,
		purger:          purger,
		swapAllowance:   swapAllowance,
		stopwords:       stopwords,
		observer:        observer,
		postingCap:      config.PostingCap,
		pause:           config.Pause,
	}
}
