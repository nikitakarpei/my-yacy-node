// Package remoteindexadmission admits what peers send to this node's remote
// index: postings over transferRWI and their URL metadata over transferURL. A
// later version replaces the stored one and its postings, within an hourly limit
// per sender network. A posting joins a rooted version by proof, an unrooted one
// from the network that sent its metadata, and waits in escrow for a later one.
package remoteindexadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type PostingReceiver interface {
	Receive(
		ctx context.Context,
		origin rwiorigin.Origin,
		receivedPagePostings []yacymodel.PagePostings,
	) (PostingReceipt, error)
}

type URLMetadataReceiver interface {
	Receive(
		ctx context.Context,
		sender rwiorigin.Origin,
		metadata []yacymodel.URLMetadata,
	) (URLMetadataReceipt, error)
}

type Escrow interface {
	IsFull(ctx context.Context) (bool, error)
	Hold(
		tx *vault.Txn,
		origin rwiorigin.Origin,
		postingAlone yacymodel.PostingAlone,
		version pageversion.Version,
	) error
	PostingsWaitingFor(
		ctx context.Context,
		tx *vault.Txn,
		url yacymodel.URLHash,
	) ([]rwiescrow.WaitingPosting, error)
	Release(tx *vault.Txn, waiting rwiescrow.WaitingPosting) error
}

type Stopwords interface {
	Contains(word yacymodel.Hash, language yacymodel.Language) bool
}

type Observer interface {
	ObserveReceivedPostings(outcome PostingOutcome, postings int)
	ObserveReleasedPostings(outcome PostingOutcome, postings int)
	ObserveURLMetadata(outcome URLMetadataOutcome, records int)
}

type PostingReceipt struct {
	Busy       bool
	Pause      time.Duration
	UnknownURL []yacymodel.URLHash
}

type URLMetadataReceipt struct {
	Busy      bool
	KnownURLs int
}

type SwapLimit struct {
	FloorPerHour int
	SharePerHour float64
}

type Config struct {
	PostingCap int
	Pause      time.Duration
	SwapLimit  SwapLimit
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	v *vault.Vault,
	urls urlmeta.URLDirectory,
	evictor urlmeta.URLEvictor,
	urlWriter urlmeta.URLMetadataWriter,
	escrow Escrow,
	postings rwipostings.PostingIndex,
	postingWriter rwipostings.PostingWriter,
	stopwords Stopwords,
	observer Observer,
	now func() time.Time,
	config Config,
) (PostingAdmission, URLMetadataIntake, error) {
	windows, err := registerSwapWindows(v)
	if err != nil {
		return PostingAdmission{}, URLMetadataIntake{}, err
	}

	placer := postingPlacer{postings: postings, writer: postingWriter}
	postingAdmission := PostingAdmission{
		vault:      v,
		urls:       urls,
		escrow:     escrow,
		placer:     placer,
		stopwords:  stopwords,
		observer:   observer,
		now:        now,
		postingCap: config.PostingCap,
		pause:      config.Pause,
	}
	urlMetadataIntake := URLMetadataIntake{
		vault:     v,
		urls:      urls,
		evictor:   evictor,
		urlWriter: urlWriter,
		swapAllowance: swapAllowance{
			vault:   v,
			windows: windows,
			urls:    urls,
			now:     now,
			limit:   config.SwapLimit,
		},
		releaser: waitingPostingReleaser{
			escrow:   escrow,
			placer:   placer,
			observer: observer,
		},
		observer: observer,
		now:      now,
	}

	return postingAdmission, urlMetadataIntake, nil
}
