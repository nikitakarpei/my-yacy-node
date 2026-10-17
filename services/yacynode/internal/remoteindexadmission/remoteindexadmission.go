// Package remoteindexadmission admits what peers send to this node's remote
// index: postings over transferRWI and their URL metadata over transferURL.
// URL metadata of a later version replaces the stored metadata and every
// posting of its URL in one transaction, within an hourly limit per sender
// network. A posting of the version the stored metadata describes joins the
// index when the network that sent the metadata sends its missing word; a
// posting of a later version waits in escrow for its metadata, and any other
// posting is dropped.
package remoteindexadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type PostingReceiver interface {
	Receive(
		ctx context.Context,
		origin rwiorigin.Origin,
		postings []yacymodel.RWIPosting,
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
	IsFull(tx *vault.Txn) (bool, error)
	Hold(tx *vault.Txn, origin rwiorigin.Origin, posting yacymodel.RWIPosting) error
	PostingsWaitingFor(tx *vault.Txn, url yacymodel.URLHash) ([]rwiescrow.WaitingPosting, error)
	Release(tx *vault.Txn, posting yacymodel.RWIPosting) error
}

type Stopwords interface {
	Contains(word yacymodel.Hash, language yacymodel.Language) bool
}

type Observer interface {
	ObserveReceivedPostings(outcome ReceivedPostingOutcome, postings int)
	ObserveReleasedPostings(placement PostingPlacement, postings int)
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

	postingAdmission := PostingAdmission{
		vault:         v,
		urls:          urls,
		escrow:        escrow,
		postings:      postings,
		postingWriter: postingWriter,
		stopwords:     stopwords,
		observer:      observer,
		now:           now,
		postingCap:    config.PostingCap,
		pause:         config.Pause,
	}

	return postingAdmission, URLMetadataIntake{
		vault:     v,
		urls:      urls,
		evictor:   evictor,
		urlWriter: urlWriter,
		swapAllowance: swapAllowance{
			windows: windows,
			urls:    urls,
			now:     now,
			limit:   config.SwapLimit,
		},
		postings: postingAdmission,
		observer: observer,
		now:      now,
	}, nil
}
