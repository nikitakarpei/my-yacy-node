package urlpostingpurge

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type URLPostingPurge struct {
	purger   rwipostings.PostingPurger
	observer Observer
}

func (p URLPostingPurge) URLPurged(tx *vault.Txn, url yacymodel.URLHash) error {
	amountOfPostings, err := p.purger.PurgePostingsOf(tx, url)
	if err != nil {
		return fmt.Errorf("purge postings of purged url: %w", err)
	}
	p.observer.ObservePostingsPurgedWithURL(tx, url, amountOfPostings)

	return nil
}
