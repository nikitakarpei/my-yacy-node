// Package pageadmission takes in the reverse word index of a page this node
// crawled. The RWI of the page is everything the node knows about that URL, so
// it replaces everything the node held for it: one transaction purges the URL,
// stores the metadata the crawl found, and admits every posting of the page.
package pageadmission

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type PageReceiver interface {
	Receive(ctx context.Context, page pagerwi.PageRWI) (Receipt, error)
}

type Receipt struct {
	Busy  bool
	Pause time.Duration
}

type Config struct {
	Pause time.Duration
}

func Open(
	v *vault.Vault,
	urls urlmeta.URLEvictor,
	metadata urlmeta.URLMetadataAdmitter,
	postings rwipostings.PostingAdmitter,
	config Config,
) PageReceiver {
	return pageAdmission{
		vault:    v,
		urls:     urls,
		metadata: metadata,
		postings: postings,
		pause:    config.Pause,
	}
}
