package provensearch_test

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/provensearch"
)

type corpus struct {
	tree  indextree.Tree
	words []indextree.Word
	hits  map[indextree.Key][]byte
}

func corpusOf(seed uint64, amountOfURLs, amountOfWords int) corpus {
	random := rand.New(rand.NewPCG(seed, seed))
	words := make([]indextree.Word, amountOfWords)
	for index := range words {
		words[index][0] = byte(index)
		words[index][1] = byte(random.Uint32())
	}
	postings := map[indextree.Key][]byte{}
	var leaves []indextree.Leaf
	for range amountOfURLs {
		var url indextree.URL
		for position := range url {
			url[position] = byte(random.Uint32())
		}
		for index, word := range words {
			if random.Float64() < 1/float64(index+1) {
				posting := provensearch.PostingOf(provensearch.Posting{Hits: uint16(random.IntN(400)), InTitle: random.IntN(8) == 0, Day: 20000})
				key := indextree.KeyOf(word, url)
				postings[key] = posting
				leaves = append(leaves, indextree.Leaf{Key: key, Value: posting})
			}
		}
	}
	tree, _ := indextree.EmptyTree(provensearch.SummaryOf).Updated(indextree.Changes{Stored: leaves})

	return corpus{tree: tree, words: words, hits: postings}
}

func (c corpus) expectedResults(query provensearch.Query) []provensearch.Result {
	var results []provensearch.Result
	seen := map[indextree.URL]bool{}
	for key := range c.hits {
		url := key.URL()
		if key.Word() != query.Words[0] || seen[url] {
			continue
		}
		seen[url] = true
		if score, present := c.scoreOf(url, query.Words); present {
			results = append(results, provensearch.Result{URL: url, Score: score})
		}
	}
	slices.SortFunc(results, func(a, b provensearch.Result) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}

			return 1
		}

		return slices.Compare(a.URL[:], b.URL[:])
	})

	return results[:min(len(results), query.Amount)]
}

func (c corpus) scoreOf(url indextree.URL, words []indextree.Word) (uint64, bool) {
	var score uint64
	total := c.tree.Summary().Postings
	for _, word := range words {
		posting, found := c.hits[indextree.KeyOf(word, url)]
		if !found {
			return 0, false
		}
		score += weightOf(c.amountOfPostingsOf(word), total) * uint64(provensearch.SummaryOf(posting).TopImpact)
	}

	return score, true
}

func (c corpus) amountOfPostingsOf(word indextree.Word) uint64 {
	var amount uint64
	for key := range c.hits {
		if key.Word() == word {
			amount++
		}
	}

	return amount
}

func weightOf(amountOfWordPostings, amountOfPostings uint64) uint64 {
	weight := uint64(1)
	for ratio := amountOfPostings / max(amountOfWordPostings, 1); ratio > 0; ratio >>= 1 {
		weight++
	}

	return weight
}

func TestACompleteAnswerHoldsTheTopResults(t *testing.T) {
	corpus := corpusOf(7, 3000, 6)
	for _, words := range [][]indextree.Word{
		{corpus.words[0]},
		{corpus.words[2]},
		{corpus.words[1], corpus.words[3]},
		{corpus.words[0], corpus.words[1], corpus.words[2]},
	} {
		query := provensearch.Query{Words: words, Amount: 10}
		answer := provensearch.AnswerFrom(corpus.tree, query, 100000)
		if !answer.Verdict.Complete {
			t.Fatalf("answer for %d words is not complete", len(words))
		}
		if !slices.Equal(answer.Verdict.Results, corpus.expectedResults(query)) {
			t.Fatalf("answer for %d words holds other results", len(words))
		}
		verdict, err := provensearch.VerifiedVerdictOf(answer.Proof, query, corpus.tree.Root(), corpus.tree.Summary())
		if err != nil || !verdict.Complete || !slices.Equal(verdict.Results, answer.Verdict.Results) {
			t.Fatalf("verifier disagrees with the answer: %v", err)
		}
	}
}

func TestAPartialAnswerBoundsEveryMissingResult(t *testing.T) {
	corpus := corpusOf(8, 3000, 4)
	query := provensearch.Query{Words: []indextree.Word{corpus.words[0], corpus.words[1]}, Amount: 10}
	answer := provensearch.AnswerFrom(corpus.tree, query, 5)
	if answer.Verdict.Complete || !answer.Verdict.HasBound {
		t.Fatal("a five-step answer claims to be complete")
	}
	for _, expected := range corpus.expectedResults(query) {
		if !slices.Contains(answer.Verdict.Results, expected) && expected.Score > answer.Verdict.Bound {
			t.Fatalf("result scoring %d is missing above the bound %d", expected.Score, answer.Verdict.Bound)
		}
	}
}

func TestAnAbsentWordGivesACompleteEmptyAnswer(t *testing.T) {
	corpus := corpusOf(9, 500, 3)
	query := provensearch.Query{Words: []indextree.Word{corpus.words[0], {0xff, 0xff}}, Amount: 10}
	answer := provensearch.AnswerFrom(corpus.tree, query, 100)
	verdict, err := provensearch.VerifiedVerdictOf(answer.Proof, query, corpus.tree.Root(), corpus.tree.Summary())
	if err != nil || !verdict.Complete || len(verdict.Results) != 0 {
		t.Fatalf("absent word does not give a complete empty answer: %v", err)
	}
}
