package documentholders

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Listings struct {
	listings []listing
}

type listing struct {
	holder    peerdirectory.AskablePeer
	documents yacymodel.URLHashes
}

func NoListingsYet() *Listings {
	return &Listings{}
}

func (listings *Listings) WordPartitionAnswered(
	_ yacymodel.Hash,
	_ uint,
	answers []wordpartitionasks.ReplicaAnswer,
) {
	for _, answer := range answers {
		if answer.ListedDocuments.Amount() == 0 {
			continue
		}
		listings.listings = append(listings.listings, listing{
			holder:    answer.Holder,
			documents: answer.ListedDocuments,
		})
	}
}

func (listings *Listings) HoldersOf(documents yacymodel.URLHashes) Holders {
	holders := Holders{
		peersPerDocument: make(
			map[yacymodel.URLHash][]peerdirectory.AskablePeer,
			documents.Amount(),
		),
	}
	for document := range documents.All() {
		for _, listing := range listings.listings {
			if listing.documents.Contains(document) {
				holders.keep(listing.holder, document)
			}
		}
	}

	return holders
}
