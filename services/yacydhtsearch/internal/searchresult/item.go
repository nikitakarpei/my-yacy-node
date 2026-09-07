// Package searchresult holds one result a peer reported, the rule that merges
// the answers of many peers into one ranking, and the pages cut from it.
package searchresult

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Item struct {
	Hash         yacymodel.URLHash
	Address      string
	Title        string
	Description  string
	PublishedAt  yacymodel.Optional[time.Time]
	ImageAddress string
}

func ItemFrom(metadata yacymodel.URLMetadata) (Item, bool) {
	hash, err := metadata.Hash()
	if err != nil {
		return Item{}, false
	}

	return Item{
		Hash:         hash,
		Address:      metadata.Address,
		Title:        metadata.Title,
		Description:  metadata.Snippet,
		PublishedAt:  publicationInstantOf(metadata),
		ImageAddress: metadata.FaviconAddress,
	}, true
}

func publicationInstantOf(metadata yacymodel.URLMetadata) yacymodel.Optional[time.Time] {
	day, ok := metadata.Modified.Get()
	if !ok {
		day, ok = metadata.Loaded.Get()
	}
	if !ok {
		return yacymodel.None[time.Time]()
	}

	return yacymodel.Some(day.Time())
}
