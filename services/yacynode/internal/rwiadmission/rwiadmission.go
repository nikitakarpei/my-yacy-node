// Package rwiadmission decides where an inbound RWI posting lands. A posting
// whose URL metadata this node already holds joins the index at once; a posting
// whose URL is still unknown waits in escrow until the sender delivers that
// metadata. The receipt names the unknown URLs so the sender can send them.
package rwiadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type PostingReceiver interface {
	Receive(ctx context.Context, postings []yacymodel.RWIPosting) (Receipt, error)
}

type PostingHolder interface {
	Hold(tx *vault.Txn, posting yacymodel.RWIPosting) error
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

func Open(
	v *vault.Vault,
	urls urlmeta.URLDirectory,
	admitter rwipostings.PostingAdmitter,
	escrow PostingHolder,
	config Config,
) PostingReceiver {
	return postingAdmission{
		vault:    v,
		urls:     urls,
		admitter: admitter,
		escrow:   escrow,
		observer: config.Observer,

		postingCap: config.PostingCap,
		pause:      config.Pause,
	}
}
