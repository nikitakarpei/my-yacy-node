// Command indexcost measures what the accountable index costs: tree nodes written per
// stored posting, proof bytes for searches, lookups and count checks, and search time.
package main

import (
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/provensearch"
)

const (
	amountOfWords     = 5000
	amountOfDocuments = 30000
	wordsPerDocument  = 15
	bytesPerNode      = 90
	roundsBudget      = 400
)

func main() {
	random := rand.New(rand.NewPCG(42, 42))
	zipf := rand.NewZipf(random, 1.1, 1, amountOfWords-1)
	words := make([]indextree.Word, amountOfWords)
	for index := range words {
		words[index] = indextree.Word{byte(index >> 8), byte(index), byte(random.Uint32()), byte(random.Uint32())}
	}
	documentOf := func() []indextree.Leaf {
		var url indextree.URL
		for position := range url {
			url[position] = byte(random.Uint32())
		}
		chosen := map[uint64]bool{}
		leaves := make([]indextree.Leaf, 0, wordsPerDocument)
		for len(chosen) < wordsPerDocument {
			word := zipf.Uint64()
			if chosen[word] {
				continue
			}
			chosen[word] = true
			leaves = append(leaves, indextree.Leaf{
				Key:   indextree.KeyOf(words[word], url),
				Value: provensearch.PostingOf(provensearch.Posting{Hits: uint16(random.ExpFloat64() * 4), InTitle: random.IntN(8) == 0, Day: uint16(19000 + random.IntN(1000))}),
			})
		}

		return leaves
	}
	postingsOf := func(amount int) []indextree.Leaf {
		var leaves []indextree.Leaf
		for len(leaves) < amount {
			leaves = append(leaves, documentOf()...)
		}

		return leaves[:amount]
	}
	tree := indextree.EmptyTree(provensearch.SummaryOf)
	for range amountOfDocuments / 1000 {
		tree, _ = tree.Updated(indextree.Changes{Stored: postingsOf(1000 * wordsPerDocument)})
	}
	fmt.Fprintf(os.Stdout, "tree: %d postings, %d documents of %d words drawn from %d words (Zipf 1.1)\n\n",
		tree.Summary().Live, amountOfDocuments, wordsPerDocument, amountOfWords)
	fmt.Fprintln(os.Stdout, "| postings per seal | nodes written per posting | bytes written per posting |")
	fmt.Fprintln(os.Stdout, "|---|---|---|")
	for _, batchSize := range []int{10, 100, 1000, 10000} {
		_, written := tree.Updated(indextree.Changes{Stored: postingsOf(batchSize)})
		perPosting := float64(written) / float64(batchSize)
		fmt.Fprintf(os.Stdout, "| %d | %.1f | %.0f |\n", batchSize, perPosting, perPosting*bytesPerNode)
	}
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "| request | complete | proof bytes | time |")
	fmt.Fprintln(os.Stdout, "|---|---|---|---|")
	requests := []struct {
		name  string
		query provensearch.Query
	}{
		{"top 10, most common word", provensearch.Query{Words: words[:1], Amount: 10}},
		{"top 10, word of rank 50", provensearch.Query{Words: words[50:51], Amount: 10}},
		{"top 10, word of rank 2000", provensearch.Query{Words: words[2000:2001], Amount: 10}},
		{"top 10, words of rank 1 and 2", provensearch.Query{Words: words[1:3], Amount: 10}},
		{"top 10, words of rank 3 and 40", provensearch.Query{Words: []indextree.Word{words[3], words[40]}, Amount: 10}},
		{"top 10, words of rank 20 and 300", provensearch.Query{Words: []indextree.Word{words[20], words[300]}, Amount: 10}},
		{"top 10, words of rank 1, 2 and 3", provensearch.Query{Words: words[1:4], Amount: 10}},
		{"top 10, word 5 without word 1", provensearch.Query{Words: words[5:6], Excluded: words[1:2], Amount: 10}},
	}
	for _, request := range requests {
		started := time.Now()
		answer := provensearch.AnswerFrom(tree, request.query, roundsBudget)
		elapsed := time.Since(started)
		fmt.Fprintf(os.Stdout, "| %s | %t | %d | %s |\n",
			request.name, answer.Verdict.Complete, len(answer.Proof.Bytes()), elapsed.Round(time.Millisecond))
	}
	lookup := tree.Prover()
	lookup.RevealLookupOf(postingsOf(1)[0].Key)
	fmt.Fprintf(os.Stdout, "| lookup of one key | - | %d | - |\n", len(lookup.Proof().Bytes()))
}
