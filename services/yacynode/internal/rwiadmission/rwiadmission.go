// Package rwiadmission decides where an inbound RWI posting lands. A stopword
// posting is dropped. A posting whose URL is unknown, or whose stored metadata
// describes an older page, waits in escrow; any other posting is checked against
// the page version indexed for its URL. The receipt names the URLs of waiting
// postings so the sender can send their metadata.
package rwiadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpageversion"
)

type PostingReceiver interface {
	Receive(
		ctx context.Context,
		origin rwiorigin.Origin,
		postings []yacymodel.RWIPosting,
	) (Receipt, error)
}

type PostingHolder interface {
	AtCapacity(ctx context.Context) (bool, error)
	Hold(tx *vault.Txn, origin rwiorigin.Origin, posting yacymodel.RWIPosting) error
}

type PageVersions interface {
	Admit(
		tx *vault.Txn,
		origin rwiorigin.Origin,
		posting yacymodel.RWIPosting,
		pageVersion yacymodel.MicroDate,
	) (urlpageversion.Verdict, error)
}

type Stopwords interface {
	Contains(word yacymodel.Hash, language yacymodel.Language) bool
}

type RefusalReason string

const (
	RefusalTooManyPostings RefusalReason = "too_many_postings"
	RefusalStorageFull     RefusalReason = "storage_full"
	RefusalEscrowFull      RefusalReason = "escrow_full"
)

type RefusalObserver interface {
	ObserveRefused(reason RefusalReason, postings int)
}

type Receipt struct {
	Busy       bool
	Pause      time.Duration
	UnknownURL []yacymodel.URLHash
}

type Config struct {
	PostingCap int
	Pause      time.Duration
	Observer   RefusalObserver
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	v *vault.Vault,
	urls urlmeta.URLDirectory,
	versions PageVersions,
	escrow PostingHolder,
	stopwords Stopwords,
	config Config,
) PostingReceiver {
	return postingAdmission{
		vault:     v,
		urls:      urls,
		versions:  versions,
		escrow:    escrow,
		stopwords: stopwords,
		observer:  config.Observer,

		postingCap: config.PostingCap,
		pause:      config.Pause,
	}
}
