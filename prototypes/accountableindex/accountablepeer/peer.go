// Package accountablepeer is the smallest peer that serves verified answers: it collects
// admitted and removed postings, and at each seal evicts the stalest URLs above its quota,
// applies the changes, and signs a head over the new tree. Searches and lookups read only the
// sealed tree and carry proofs against its head.
package accountablepeer

import (
	"crypto/ed25519"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indexhead"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/provensearch"
)

type Identity struct {
	Network string
	Name    string
	Key     ed25519.PrivateKey
}

type Peer struct {
	keysByURL map[indextree.URL][]indextree.Key
	dayByURL  map[indextree.URL]uint16
	latest    indexhead.SignedHead
	identity  Identity
	pending   indextree.Changes
	tree      indextree.Tree
	quota     uint64
}

func New(identity Identity, quota uint64) *Peer {
	return &Peer{
		keysByURL: map[indextree.URL][]indextree.Key{},
		dayByURL:  map[indextree.URL]uint16{},
		identity:  identity,
		tree:      indextree.EmptyTree(provensearch.SummaryOf),
		quota:     quota,
	}
}

func (p *Peer) PublicKey() ed25519.PublicKey {
	public, _ := p.identity.Key.Public().(ed25519.PublicKey)

	return public
}

func (p *Peer) Admit(leaves []indextree.Leaf) {
	for _, leaf := range leaves {
		url := leaf.Key.URL()
		p.keysByURL[url] = append(p.keysByURL[url], leaf.Key)
		if posting, isPosting := provensearch.PostingFrom(leaf.Value); isPosting {
			p.dayByURL[url] = posting.Day
		}
	}
	p.pending.Stored = append(p.pending.Stored, leaves...)
}

func (p *Peer) RemoveURL(url indextree.URL) {
	p.pending.Removed = append(p.pending.Removed, p.keysByURL[url]...)
	delete(p.keysByURL, url)
	delete(p.dayByURL, url)
}

func (p *Peer) Seal() indexhead.SignedHead {
	p.evictAboveQuota()
	p.tree, _ = p.tree.Updated(p.pending)
	p.pending = indextree.Changes{}
	p.latest = indexhead.SignedHeadOf(indexhead.Head{
		Network:  p.identity.Network,
		Peer:     p.identity.Name,
		Sequence: p.latest.Head.Sequence + 1,
		Root:     p.tree.Root(),
		Summary:  p.tree.Summary(),
	}, p.identity.Key)

	return p.latest
}

func (p *Peer) evictAboveQuota() {
	var amountOfPostings uint64
	urls := make([]indextree.URL, 0, len(p.keysByURL))
	for url, keys := range p.keysByURL {
		amountOfPostings += uint64(len(keys))
		urls = append(urls, url)
	}
	slices.SortFunc(urls, func(a, b indextree.URL) int {
		if p.dayByURL[a] != p.dayByURL[b] {
			return int(p.dayByURL[a]) - int(p.dayByURL[b])
		}

		return slices.Compare(a[:], b[:])
	})
	for _, url := range urls {
		if amountOfPostings <= p.quota {
			return
		}
		amountOfPostings -= uint64(len(p.keysByURL[url]))
		p.RemoveURL(url)
	}
}

func (p *Peer) Search(query provensearch.Query, amountOfRounds int) (indexhead.SignedHead, provensearch.Answer) {
	return p.latest, provensearch.AnswerFrom(p.tree, query, amountOfRounds)
}

func (p *Peer) Lookup(keys []indextree.Key) (indexhead.SignedHead, indextree.Proof) {
	prover := p.tree.Prover()
	for _, key := range keys {
		prover.RevealLookupOf(key)
	}

	return p.latest, prover.Proof()
}
