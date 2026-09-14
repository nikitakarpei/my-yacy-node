package urlpostingpurge

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

type URLPostingPurge struct {
	references urlreferences.ReferenceQuery
	purger     rwipostings.PostingPurger
	observer   Observer
}

func (p URLPostingPurge) URLStored(
	*vault.Txn,
	yacymodel.URLHash,
	yacymodel.Optional[yacymodel.CalendarDay],
) error {
	return nil
}

func (p URLPostingPurge) URLPurged(tx *vault.Txn, url yacymodel.URLHash) error {
	words, err := p.references.WordsReferencing(tx, url)
	if err != nil {
		return fmt.Errorf("words referencing purged url: %w", err)
	}

	amountOfPostings := 0
	for _, word := range words {
		wasPurged, err := p.purger.PurgePosting(tx, word, url)
		if err != nil {
			return fmt.Errorf("purge posting of purged url: %w", err)
		}
		if wasPurged {
			amountOfPostings++
		}
	}
	p.observer.ObservePostingsPurgedWithURL(url, amountOfPostings)

	return nil
}
