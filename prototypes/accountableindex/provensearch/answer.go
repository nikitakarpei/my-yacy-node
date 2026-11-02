package provensearch

import (
	"bytes"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/accountableindex/indextree"
)

type Answer struct {
	Proof   indextree.Proof
	Verdict Verdict
}

func AnswerFrom(tree indextree.Tree, query Query, amountOfRounds int) Answer {
	prover := tree.Prover()
	for _, word := range query.Words {
		prover.RevealRegionOf(word)
	}
	amountOfPostings := tree.Summary().Live
	proof := prover.Proof()
	verdict := VerdictOf(proof, query, amountOfPostings)
	for range amountOfRounds {
		if verdict.Complete || !expandedTowardCompletion(prover, examinationOf(proof, query, amountOfPostings)) {
			break
		}
		proof = prover.Proof()
		verdict = VerdictOf(proof, query, amountOfPostings)
	}

	return Answer{Proof: proof, Verdict: verdict}
}

func expandedTowardCompletion(prover *indextree.Prover, examination examination) bool {
	unseenBound, hasUnseen := examination.boundOfUnseen()
	resolving := false
	for _, url := range examination.candidateURLs() {
		if _, upperBound, resolved, _ := examination.scoreOf(url); !resolved && (!hasUnseen || upperBound >= unseenBound) {
			for _, word := range append(slices.Clone(examination.query.Words), examination.query.Excluded...) {
				prover.RevealLookupOf(indextree.KeyOf(word, url))
			}
			resolving = true
		}
	}
	if resolving || !hasUnseen {
		return resolving
	}
	stubs := examination.drivingStubsInOrder()
	for _, stub := range stubs[:max(1, min(len(stubs), examination.query.Amount))] {
		prover.Expand(stub)
	}

	return true
}

func (e examination) drivingStubsInOrder() []*indextree.ProofNode {
	var driving wordExamination
	var drivingContribution uint64
	for _, examined := range e.examined {
		if contribution := examined.weight * uint64(examined.frontierTop); contribution >= drivingContribution {
			driving, drivingContribution = examined, contribution
		}
	}
	var stubs []*indextree.ProofNode
	for _, stub := range driving.region.Stubs() {
		if stub.Summary.Live > 0 {
			stubs = append(stubs, stub)
		}
	}
	slices.SortFunc(stubs, func(a, b *indextree.ProofNode) int {
		if a.Summary.TopImpact != b.Summary.TopImpact {
			return int(b.Summary.TopImpact) - int(a.Summary.TopImpact)
		}

		return bytes.Compare(a.Prefix[:], b.Prefix[:])
	})

	return stubs
}
