// Package rwiadmission decides where an inbound RWI posting lands. A posting
// whose URL metadata this node already holds joins the index at once; a posting
// whose URL is still unknown waits in escrow until the sender delivers that
// metadata. The receipt names the unknown URLs so the sender can send them.
package rwiadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type PostingReceiver interface {
	Receive(ctx context.Context, entries []yacymodel.RWIPosting) (Receipt, error)
}

type PostingHolder interface {
	Hold(tx *vault.Txn, posting yacymodel.RWIPosting) error
}

type Receipt struct {
	Busy       bool
	TooLarge   bool
	Pause      time.Duration
	UnknownURL []yacymodel.URLHash
}

type Config struct {
	BatchCap int
	Pause    time.Duration
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
		batchCap: config.BatchCap,
		pause:    config.Pause,
	}
}
