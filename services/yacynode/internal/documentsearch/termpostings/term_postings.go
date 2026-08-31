// Package termpostings reads the posting index for one search: it collects, for
// each term, the documents that hold the term and how often the term appears in
// them, and it collects the documents that hold a term the search excludes. It
// is the only reader of words in a search pass, and it holds the bound on how
// many postings of one term the node keeps in memory.
package termpostings

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Match struct {
	PostingPerDocument map[yacymodel.URLHash]Posting
	PostingsHeld       int
}

func PostingsHeldPerTermOf(matches map[yacymodel.Hash]Match) map[yacymodel.Hash]int {
	held := make(map[yacymodel.Hash]int, len(matches))
	for term, match := range matches {
		held[term] = match.PostingsHeld
	}

	return held
}

type TermPostings interface {
	MatchesFor(
		ctx context.Context,
		tx *vault.Txn,
		terms []yacymodel.Hash,
		filter postingfilter.Filter,
	) (map[yacymodel.Hash]Match, error)
	DocumentsContaining(
		ctx context.Context,
		tx *vault.Txn,
		terms []yacymodel.Hash,
	) (map[yacymodel.URLHash]struct{}, error)
}

func New(index rwipostings.PostingIndex, maxPostingsPerTerm int) TermPostings {
	return termPostings{index: index, maxPostingsPerTerm: maxPostingsPerTerm}
}

type termPostings struct {
	index              rwipostings.PostingIndex
	maxPostingsPerTerm int
}

func (t termPostings) MatchesFor(
	ctx context.Context,
	tx *vault.Txn,
	terms []yacymodel.Hash,
	filter postingfilter.Filter,
) (map[yacymodel.Hash]Match, error) {
	matches := make(map[yacymodel.Hash]Match, len(terms))
	for _, term := range terms {
		match, err := t.matchOf(ctx, tx, term, filter)
		if err != nil {
			return nil, err
		}
		matches[term] = match
	}

	return matches, nil
}

func (t termPostings) matchOf(
	ctx context.Context,
	tx *vault.Txn,
	term yacymodel.Hash,
	filter postingfilter.Filter,
) (Match, error) {
	// The per-term cap keeps the most frequent postings rather than the first
	// scanned; an exact join under a memory bound would instead pivot on the rarest term.
	frequentPostings := mostFrequentPostings{maxPostings: t.maxPostingsPerTerm}
	var held int
	err := t.index.ScanWord(ctx, tx, term, func(posting yacymodel.RWIPosting) (bool, error) {
		held++
		if !filter.Accepts(posting) {
			return true, nil
		}
		frequentPostings.consider(Posting{
			DocumentHash: posting.URLHash,
			Occurrences:  posting.Hits,
			TextPosition: posting.TextPosition,
		})

		return true, nil
	})
	if err != nil {
		return Match{}, fmt.Errorf("scan term: %w", err)
	}

	return Match{
		PostingPerDocument: postingPerDocument(frequentPostings.postings),
		PostingsHeld:       held,
	}, nil
}

func postingPerDocument(postings []Posting) map[yacymodel.URLHash]Posting {
	byDocument := make(map[yacymodel.URLHash]Posting, len(postings))
	for _, posting := range postings {
		byDocument[posting.DocumentHash] = posting
	}

	return byDocument
}

func (t termPostings) DocumentsContaining(
	ctx context.Context,
	tx *vault.Txn,
	terms []yacymodel.Hash,
) (map[yacymodel.URLHash]struct{}, error) {
	documents := make(map[yacymodel.URLHash]struct{})
	for _, term := range terms {
		err := t.index.ScanWord(
			ctx,
			tx,
			term,
			func(posting yacymodel.RWIPosting) (bool, error) {
				documents[posting.URLHash] = struct{}{}

				return true, nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("scan term: %w", err)
		}
	}

	return documents, nil
}
