// Package rwiadmission decides where an inbound RWI posting lands. A posting
// whose URL metadata this node already holds joins the index at once; a posting
// whose URL is still unknown waits in escrow until the sender delivers that
// metadata. The receipt names the unknown URLs so the sender can send them.
// Postings a peer sends join the postings this node already holds for their
// URLs; the postings of a page this node crawled itself replace every posting
// the page had before, so a word the page lost keeps no posting behind.
package rwiadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

type PostingReceiver interface {
	Receive(ctx context.Context, entries []yacymodel.RWIPosting) (Receipt, error)
}

type PagePostingReceiver interface {
	ReceiveEveryPostingOfPage(
		ctx context.Context,
		pageURL yacymodel.URLHash,
		postings []yacymodel.RWIPosting,
	) (Receipt, error)
}

type PostingHolder interface {
	Hold(tx *vault.Txn, posting yacymodel.RWIPosting) error
}

type RefusalReason string

const (
	RefusalRequestTooLarge RefusalReason = "request_too_large"
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
	Pause    time.Duration
	Refusals RefusalObserver
}

//nolint:revive // argument-limit: each argument is a distinct collection the admission writes to or reads; bundling them would invent a hollow type
func Open(
	v *vault.Vault,
	urls urlmeta.URLDirectory,
	admitter rwipostings.PostingAdmitter,
	purger rwipostings.PostingPurger,
	references urlreferences.ReferenceQuery,
	escrow PostingHolder,
	config Config,
) (PostingReceiver, PagePostingReceiver) {
	admission := postingAdmission{
		vault:      v,
		urls:       urls,
		admitter:   admitter,
		purger:     purger,
		references: references,
		escrow:     escrow,
		observer:   config.Refusals,
		pause:      config.Pause,
	}

	return admission, admission
}
