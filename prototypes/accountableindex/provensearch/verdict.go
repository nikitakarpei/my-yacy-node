package provensearch

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

var ErrProofOfOtherHead = errors.New("proof does not lead to the head")

type Query struct {
	Words    []indextree.Word
	Excluded []indextree.Word
	Amount   int
	Language Language
}

type Result struct {
	URL   indextree.URL
	Score uint64
}

type Verdict struct {
	Results  []Result
	Bound    uint64
	Complete bool
	HasBound bool
}

func VerifiedProof(proof indextree.Proof, root indextree.Hash, summary indextree.Summary) error {
	provenRoot, provenSummary, err := proof.VerifiedRoot(SummaryOf)
	if err != nil {
		return fmt.Errorf("verify proof: %w", err)
	}
	if provenRoot != root || provenSummary != summary {
		return ErrProofOfOtherHead
	}

	return nil
}

func VerifiedVerdictOf(
	proof indextree.Proof,
	query Query,
	root indextree.Hash,
	summary indextree.Summary,
) (Verdict, error) {
	if err := VerifiedProof(proof, root, summary); err != nil {
		return Verdict{}, err
	}

	return VerdictOf(proof, query, summary.Postings), nil
}

func VerdictOf(proof indextree.Proof, query Query, amountOfPostings uint64) Verdict {
	examination := examinationOf(proof, query, amountOfPostings)
	if examination.hasAbsentWord {
		return Verdict{Complete: true}
	}
	if examination.hasUnprovenWord {
		return Verdict{Bound: math.MaxUint64, HasBound: true}
	}
	results, bound, hasBound := examination.resultsAndBound()
	verdict := Verdict{Bound: bound, HasBound: hasBound}
	verdict.Complete = !hasBound || (len(results) >= query.Amount && examination.outranks(results[query.Amount-1], bound))
	verdict.Results = results[:min(len(results), query.Amount)]

	return verdict
}

func (e examination) outranks(last Result, bound uint64) bool {
	if last.Score != bound {
		return last.Score > bound
	}
	for _, url := range e.unresolvedURLs() {
		if bytes.Compare(url[:], last.URL[:]) < 0 {
			return false
		}
	}
	if unseenBound, hasUnseen := e.boundOfUnseen(); !hasUnseen || unseenBound < last.Score {
		return true
	}
	for _, examined := range e.examined {
		if examined.tiesOnlyAfter(last.URL) {
			return true
		}
	}

	return false
}

func (w wordExamination) tiesOnlyAfter(url indextree.URL) bool {
	for _, stub := range w.region.Stubs() {
		lowest := stub.LowestURL()
		if stub.Summary.Postings > 0 && stub.Summary.TopImpact == w.frontierTop && bytes.Compare(lowest[:], url[:]) <= 0 {
			return false
		}
	}

	return true
}

func (e examination) unresolvedURLs() []indextree.URL {
	var urls []indextree.URL
	for _, url := range e.candidateURLs() {
		if _, _, resolved, _ := e.scoreOf(url); !resolved {
			urls = append(urls, url)
		}
	}

	return urls
}

type wordExamination struct {
	region      *indextree.ProofNode
	frontierTop uint16
	weight      uint64
	hasFrontier bool
}

type examination struct {
	proof           indextree.Proof
	query           Query
	examined        []wordExamination
	hasAbsentWord   bool
	hasUnprovenWord bool
}

func examinationOf(proof indextree.Proof, query Query, amountOfPostings uint64) examination {
	result := examination{proof: proof, query: query, examined: make([]wordExamination, len(query.Words))}
	for index, word := range query.Words {
		region, presence := proof.RegionOf(word)
		switch {
		case presence == indextree.Unproven:
			result.hasUnprovenWord = true
		case presence == indextree.Absent || region.Summary.Postings == 0:
			result.hasAbsentWord = true
		default:
			result.examined[index] = wordExaminationOf(region, amountOfPostings)
		}
	}

	return result
}

func wordExaminationOf(region *indextree.ProofNode, amountOfPostings uint64) wordExamination {
	examined := wordExamination{region: region, weight: weightOf(region.Summary.Postings, amountOfPostings)}
	for _, stub := range region.Stubs() {
		if stub.Summary.Postings > 0 {
			examined.hasFrontier = true
			examined.frontierTop = max(examined.frontierTop, stub.Summary.TopImpact)
		}
	}

	return examined
}

func weightOf(amountOfWordPostings, amountOfPostings uint64) uint64 {
	return uint64(bits.Len64(amountOfPostings/max(amountOfWordPostings, 1))) + 1
}

func (e examination) resultsAndBound() (results []Result, bound uint64, hasBound bool) {
	bound, hasBound = e.boundOfUnseen()
	for _, url := range e.candidateURLs() {
		score, upperBound, resolved, present := e.scoreOf(url)
		switch {
		case !resolved:
			bound, hasBound = max(bound, upperBound), true
		case present:
			results = append(results, Result{URL: url, Score: score})
		}
	}
	slices.SortFunc(results, func(a, b Result) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}

			return 1
		}

		return bytes.Compare(a.URL[:], b.URL[:])
	})

	return results, bound, hasBound
}

func (e examination) boundOfUnseen() (uint64, bool) {
	var bound uint64
	for _, examined := range e.examined {
		if !examined.hasFrontier {
			return 0, false
		}
		bound += examined.weight * uint64(examined.frontierTop)
	}

	return bound, true
}

func (e examination) passesFilters(leaf *indextree.Leaf) bool {
	if leaf == nil || e.query.Language == AnyLanguage {
		return true
	}
	posting, isPosting := PostingFrom(leaf.Value)

	return isPosting && posting.Language == e.query.Language
}

func (e examination) candidateURLs() []indextree.URL {
	seen := map[indextree.URL]bool{}
	var urls []indextree.URL
	for _, examined := range e.examined {
		for _, leaf := range examined.region.RevealedLeaves() {
			url := leaf.Leaf.Key.URL()
			if leaf.Summary.Postings == 1 && !seen[url] {
				seen[url] = true
				urls = append(urls, url)
			}
		}
	}

	return urls
}

func (e examination) scoreOf(url indextree.URL) (score, upperBound uint64, resolved, present bool) {
	resolved, present = true, true
	for index, word := range e.query.Words {
		found, presence := e.proof.Lookup(indextree.KeyOf(word, url))
		switch {
		case presence == indextree.Unproven:
			resolved = false
			upperBound += e.examined[index].weight * uint64(found.Summary.TopImpact)
		case presence == indextree.Absent || found.Summary.Postings == 0:
			return 0, 0, true, false
		case found.Leaf == nil && e.query.Language != AnyLanguage:
			resolved = false
			upperBound += e.examined[index].weight * uint64(found.Summary.TopImpact)
		case !e.passesFilters(found.Leaf):
			return 0, 0, true, false
		default:
			contribution := e.examined[index].weight * uint64(found.Summary.TopImpact)
			score += contribution
			upperBound += contribution
		}
	}
	for _, word := range e.query.Excluded {
		found, presence := e.proof.Lookup(indextree.KeyOf(word, url))
		switch {
		case presence == indextree.Unproven:
			resolved = false
		case presence == indextree.Present && found.Summary.Postings == 1:
			return 0, 0, true, false
		}
	}

	return score, upperBound, resolved, present
}
