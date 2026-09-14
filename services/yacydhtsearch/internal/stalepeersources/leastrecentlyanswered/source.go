// Package leastrecentlyanswered ranks the peers that answered longest ago as
// the stalest, so a directory keeps the peers that still reply.
package leastrecentlyanswered

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Source struct{}

func New() Source {
	return Source{}
}

func (Source) StalestPeers(
	known []peerdirectory.KnownPeer,
	limit int,
) []yacymodel.Hash {
	if limit <= 0 {
		return nil
	}

	ranked := slices.SortedFunc(
		slices.Values(known),
		func(a, b peerdirectory.KnownPeer) int {
			return cmp.Or(
				a.AnsweredAt.Compare(b.AnsweredAt),
				a.AdmittedAt.Compare(b.AdmittedAt),
			)
		},
	)
	stalest := make([]yacymodel.Hash, 0, min(limit, len(ranked)))
	for _, peer := range ranked[:min(limit, len(ranked))] {
		stalest = append(stalest, peer.Hash)
	}

	return stalest
}
